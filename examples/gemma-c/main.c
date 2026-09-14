/* Gemma 3 270M: BF16 weights, float activations, single-sequence KV cache.
 * Numerical inference is entirely C. Go provides tokenization, decoding and I/O.
 */
#pragma go "bridge.go"
#pragma go "tokenizer.go"
#include <stdint.h>
#include <stdio.h>

extern uintptr_t llm_load(void);
extern void llm_args(int argc, uintptr_t argv);
extern int llm_command(void);
extern void llm_finish(void);
extern uint32_t llm_size(void);
extern int llm_prompt_length(void);
extern int llm_prompt_token(int i);
extern int llm_steps(void);
extern void llm_emit(int token);
extern void llm_logits(uintptr_t address, int count);

#define D 640
#define FF 2048
#define NL 18
#define NH 4
#define HD 256
#define VOCAB 262144
#define CTX 1024

union FloatBits { uint32_t bits; float value; };
static unsigned char *model;
static uint32_t *header;
static float x[D], norm[D], projected[D], mlp[FF], gate[FF];
static float q[NH*HD], k[HD], v[HD], attended[NH*HD];
static float keys[NL*CTX*HD], values[NL*CTX*HD];
static float scores[CTX], logits[VOCAB];
static float epsilon, attention_scale;

static float frombits(uint32_t bits) {
    union FloatBits b;
    b.bits = bits;
    return b.value;
}
static float bf16(uint16_t bits) { return frombits((uint32_t)bits << 16); }
static uint16_t *weight(int layer, int field) {
    return (uint16_t *)(model + header[64 + layer*16 + field]);
}
static float rsqrt(float x) {
    union FloatBits b;
    float y;
    int i;
    b.value = x;
    b.bits = 0x5f3759df - (b.bits >> 1);
    y = b.value;
    for (i = 0; i < 4; i++) y = y * (1.5f - 0.5f*x*y*y);
    return y;
}
static float exponential(float x) {
    int n;
    double y, p;
    if (x < -80.0f) return 0.0f;
    if (x > 80.0f) x = 80.0f;
    n = (int)(x * 1.4426950408889634 + (x >= 0.0f ? 0.5 : -0.5));
    y = (double)x - n*0.6931471805599453;
    p = 1.0 + y*(1.0 + y*(0.5 + y*(1.0/6.0 + y*(1.0/24.0 + y*(1.0/120.0 + y*(1.0/720.0 + y*(1.0/5040.0 + y/40320.0)))))));
    return (float)p * frombits((uint32_t)(n + 127) << 23);
}
static float sine(double x) {
    double term, sum;
    int n;
    x -= (int)(x/6.283185307179586)*6.283185307179586;
    if (x > 3.141592653589793) x -= 6.283185307179586;
    if (x < -3.141592653589793) x += 6.283185307179586;
    term = x; sum = x;
    for (n = 1; n < 10; n++) { term *= -x*x/((2*n)*(2*n+1)); sum += term; }
    return (float)sum;
}
static float gelu(float x) {
    float t = 0.7978845608028654f*(x + 0.044715f*x*x*x);
    float th;
    if (t > 8.0f) th = 1.0f;
    else if (t < -8.0f) th = -1.0f;
    else { float e = exponential(2.0f*t); th = (e-1.0f)/(e+1.0f); }
    return 0.5f*x*(1.0f+th);
}
static void matvec(float *out, const uint16_t *w, const float *in, int rows, int cols) {
    int i, j;
    for (i = 0; i < rows; i++) {
        float sum = 0.0f;
        for (j = 0; j < cols; j++) sum += bf16(w[i*cols+j])*in[j];
        out[i] = sum;
    }
}
static void rms(float *out, const float *in, const uint16_t *w, int n) {
    int i;
    float sum = 0.0f, scale;
    for (i = 0; i < n; i++) sum += in[i]*in[i];
    scale = rsqrt(sum/n + epsilon);
    for (i = 0; i < n; i++) out[i] = in[i]*scale*(1.0f+bf16(w[i]));
}
static void rotary(float *vec, const float *freq, int pos) {
    int i;
    for (i = 0; i < HD/2; i++) {
        float a = vec[i], b = vec[i+HD/2];
        float angle = pos*freq[i];
        float s = sine(angle), c = sine((double)angle+1.5707963267948966);
        vec[i] = a*c-b*s;
        vec[i+HD/2] = b*c+a*s;
    }
}
static void forward(int token, int pos, int output) {
    int i, l, h, t;
    uint16_t *embed = (uint16_t *)(model+header[32]);
    for (i = 0; i < D; i++) x[i] = bf16(embed[token*D+i])*25.298221281347036f;
    for (l = 0; l < NL; l++) {
        int local = header[64+l*16+13];
        const float *freq = (const float *)(model+header[local ? 34 : 35]);
        int first = local && pos >= (int)header[10] ? pos-(int)header[10]+1 : 0;
        rms(norm, x, weight(l,0), D);
        matvec(q, weight(l,1), norm, NH*HD, D);
        matvec(k, weight(l,2), norm, HD, D);
        matvec(v, weight(l,3), norm, HD, D);
        for (h = 0; h < NH; h++) { rms(q+h*HD, q+h*HD, weight(l,4), HD); rotary(q+h*HD, freq, pos); }
        rms(k, k, weight(l,5), HD); rotary(k, freq, pos);
        for (i = 0; i < HD; i++) { keys[(l*CTX+pos)*HD+i]=k[i]; values[(l*CTX+pos)*HD+i]=v[i]; }
        for (h = 0; h < NH; h++) {
            float maxscore = -1.0e30f, total = 0.0f;
            for (t = first; t <= pos; t++) {
                float sum = 0.0f;
                for (i = 0; i < HD; i++) sum += q[h*HD+i]*keys[(l*CTX+t)*HD+i];
                scores[t] = sum*attention_scale;
                if (scores[t] > maxscore) maxscore = scores[t];
            }
            for (t = first; t <= pos; t++) { scores[t] = exponential(scores[t]-maxscore); total += scores[t]; }
            for (i = 0; i < HD; i++) {
                float sum = 0.0f;
                for (t = first; t <= pos; t++) sum += (scores[t]/total)*values[(l*CTX+t)*HD+i];
                attended[h*HD+i] = sum;
            }
        }
        matvec(projected, weight(l,6), attended, D, NH*HD);
        rms(norm, projected, weight(l,7), D);
        for (i = 0; i < D; i++) x[i] += norm[i];
        rms(norm, x, weight(l,8), D);
        matvec(gate, weight(l,9), norm, FF, D);
        matvec(mlp, weight(l,10), norm, FF, D);
        for (i = 0; i < FF; i++) mlp[i] *= gelu(gate[i]);
        matvec(projected, weight(l,11), mlp, D, FF);
        rms(norm, projected, weight(l,12), D);
        for (i = 0; i < D; i++) x[i] += norm[i];
    }
    if (output) {
        rms(norm, x, (uint16_t *)(model+header[33]), D);
        matvec(logits, embed, norm, VOCAB, D);
    }
}
static int valid_extent(uint32_t offset, uint32_t elements, uint32_t bytes) {
    return offset >= 2048 && offset % 2 == 0 && offset <= bytes && elements <= (bytes-offset)/2;
}
int main(int argc, char **argv) {
    int i, l, step, token, n, steps;
    uint32_t size;
    uint32_t counts[13] = {D, NH*HD*D, HD*D, HD*D, HD, HD, D*NH*HD, D, D, FF*D, FF*D, D*FF, D};
    llm_args(argc, (uintptr_t)argv);
    i = llm_command();
    if (i >= 0) return i;
    model = (unsigned char *)llm_load();
    if (!model) { fprintf(stderr, "Cannot load model/prompt\n"); return 1; }
    header = (uint32_t *)model; size = llm_size();
    if (header[0]!=0x31434d47 || header[1]!=1 || header[2]!=size || header[3]!=D || header[4]!=FF || header[5]!=NL || header[6]!=NH || header[7]!=1 || header[8]!=HD || header[9]!=VOCAB || header[10]==0 || header[10]>CTX || header[11]!=CTX) return 2;
    if (!valid_extent(header[32], VOCAB*D, size) || !valid_extent(header[33], D, size)) return 2;
    for (i=34; i<=35; i++) if (header[i]%4!=0 || !valid_extent(header[i], HD, size)) return 2;
    for (l=0; l<NL; l++) {
        for (i=0; i<13; i++) if (!valid_extent(header[64+l*16+i], counts[i], size)) return 2;
        if (header[64+l*16+13]>1) return 2;
    }
    epsilon=frombits(header[12]); attention_scale=frombits(header[13]);
    if (!(epsilon>0.0f && epsilon<1.0f && attention_scale>0.0f && attention_scale<=1.0f)) return 2;
    n=llm_prompt_length(); steps=llm_steps();
    if (n<1 || n>CTX || steps<0 || steps>CTX-n) return 2;
    for (i=0; i<n; i++) if (llm_prompt_token(i)<0 || llm_prompt_token(i)>=VOCAB) return 2;
    for (i=0; i<n; i++) forward(llm_prompt_token(i), i, i==n-1);
    llm_logits((uintptr_t)logits, VOCAB);
    for (step=0; step<steps; step++) {
        token=0;
        for (i=1; i<VOCAB; i++) if (logits[i]>logits[token]) token=i;
        llm_emit(token);
        if (token==1 || token==106 || step+1==steps) break;
        forward(token,n+step,1);
    }
    llm_finish();
    return 0;
}
