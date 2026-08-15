#include <openssl/crypto.h>
#include <openssl/sha.h>
#include <openssl/rand.h>
#include <openssl/evp.h>

int main(void) {
    unsigned long version_number = OpenSSL_version_num();
    if (version_number == 0) return 1;
    if (OpenSSL_version(0) == NULL) return 2;

    unsigned char message[3] = {'a', 'b', 'c'};
    unsigned char expected[32] = {
        0xba, 0x78, 0x16, 0xbf, 0x8f, 0x01, 0xcf, 0xea,
        0x41, 0x41, 0x40, 0xde, 0x5d, 0xae, 0x22, 0x23,
        0xb0, 0x03, 0x61, 0xa3, 0x96, 0x17, 0x7a, 0x9c,
        0xb4, 0x10, 0xff, 0x61, 0xf2, 0x00, 0x15, 0xad,
    };
    unsigned char one_shot[32];
    if (SHA256(&message[0], 3, &one_shot[0]) == NULL) return 3;
    if (CRYPTO_memcmp(&one_shot[0], &expected[0], 32) != 0) return 4;

    EVP_MD_CTX *context = EVP_MD_CTX_new();
    if (context == NULL) return 5;
    const EVP_MD *digest = EVP_sha256();
    if (digest == NULL) {
        EVP_MD_CTX_free(context);
        return 6;
    }
    if (EVP_DigestInit_ex(context, digest, NULL) != 1) {
        EVP_MD_CTX_free(context);
        return 7;
    }
    if (EVP_DigestUpdate(context, &message[0], 3) != 1) {
        EVP_MD_CTX_free(context);
        return 8;
    }
    unsigned char streamed[32];
    unsigned int streamed_size = 0;
    if (EVP_DigestFinal_ex(context, &streamed[0], &streamed_size) != 1) {
        EVP_MD_CTX_free(context);
        return 9;
    }
    EVP_MD_CTX_free(context);
    if (streamed_size != 32) return 10;
    if (CRYPTO_memcmp(&streamed[0], &one_shot[0], 32) != 0) return 11;

    unsigned char random_a[32];
    unsigned char random_b[32];
    if (RAND_bytes(&random_a[0], 32) != 1) return 12;
    if (RAND_bytes(&random_b[0], 32) != 1) return 13;
    int nonzero = 0;
    int i = 0;
    for (i = 0; i < 32; i++) {
        if (random_a[i] != 0) nonzero = 1;
    }
    if (nonzero == 0) return 14;
    if (CRYPTO_memcmp(&random_a[0], &random_b[0], 32) == 0) return 15;
    return 0;
}
