package main

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"reflect"
)

type Embedded struct {
	ID int `json:"id"`
}

type taggedPayload struct {
	Embedded
	Name    string `json:"display"`
	Empty   string `json:"empty,omitempty"`
	Count   int    `json:"count,string"`
	Values  []int  `json:"values"`
	Ignored string `json:"-"`
	hidden  string
}

type memoryWriter struct{ data []byte }

func (w *memoryWriter) Write(data []byte) (int, error) {
	w.data = append(w.data, data...)
	return len(data), nil
}

type memoryReader struct {
	data []byte
	pos  int
}

type rawHolder struct {
	Data json.RawMessage `json:"data"`
}

func (r *memoryReader) Read(data []byte) (int, error) {
	if r.pos == len(r.data) {
		return 0, nil
	}
	n := copy(data, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func main() {
	hexEncoded := hex.EncodeToString([]byte("hello"))
	if hexEncoded != "68656c6c6f" {
		print("FAIL hex encode\n")
		return
	}
	decoded, err := hex.DecodeString("68656c6c6f")
	if err != nil || string(decoded) != "hello" {
		print("FAIL hex decode\n")
		return
	}
	token := base64.RawURLEncoding.EncodeToString([]byte("hello?"))
	if token != "aGVsbG8_" {
		print("FAIL base64 encode\n")
		return
	}
	decoded, err = base64.RawURLEncoding.DecodeString(token)
	if err != nil || string(decoded) != "hello?" {
		print("FAIL base64 decode\n")
		return
	}
	malformed := []string{"!", "a", "ab$c", "ab=c"}
	for _, input := range malformed {
		if _, decodeErr := base64.RawURLEncoding.DecodeString(input); decodeErr == nil {
			print("FAIL base64 malformed\n")
			return
		}
	}
	if !json.Valid([]byte(`{"a":[1,"x",false],"u":"\u20ac"}`)) || json.Valid([]byte(`[1,]`)) {
		print("FAIL json valid\n")
		return
	}
	encoded, err := json.Marshal(map[string]any{"z": []any{1, true, nil}, "a": "x\n"})
	if err != nil || string(encoded) != `{"a":"x\n","z":[1,true,null]}` {
		print("FAIL json marshal: " + string(encoded) + "\n")
		return
	}
	pretty, err := json.MarshalIndent(map[string]any{"a": []any{1, "x"}}, "", "  ")
	if err != nil || string(pretty) != "{\n  \"a\": [\n    1,\n    \"x\"\n  ]\n}" {
		print("FAIL json indent: " + string(pretty) + "\n")
		return
	}
	prefixed, err := json.MarshalIndent(map[string]any{"empty": []any{}, "nested": map[string]any{"ok": true}}, "> ", "--")
	if err != nil || string(prefixed) != "{\n> --\"empty\": [],\n> --\"nested\": {\n> ----\"ok\": true\n> --}\n> }" {
		print("FAIL json indent prefix: " + string(prefixed) + "\n")
		return
	}
	raw := json.RawMessage(`{"ok":true}`)
	encoded, err = json.Marshal(raw)
	if err != nil || string(encoded) != string(raw) {
		print("FAIL json raw\n")
		return
	}
	typ := reflect.TypeOf(taggedPayload{})
	if typ.Kind() != reflect.Struct || typ.NumField() != 7 || typ.Field(1).Tag.Get("json") != "display" || !typ.Field(0).Anonymous {
		print("FAIL reflect tags\n")
		return
	}
	encoded, err = json.Marshal(&taggedPayload{Embedded: Embedded{ID: 7}, Name: "renvo", Count: 3, Values: []int{2, 4}, Ignored: "no", hidden: "no"})
	if err != nil || string(encoded) != `{"id":7,"display":"renvo","count":"3","values":[2,4]}` {
		print("FAIL json struct tags: " + string(encoded) + "\n")
		return
	}
	var dynamic any
	err = json.Unmarshal([]byte(`{"name":"renvo","values":[2,true,null],"unicode":"\u20ac"}`), &dynamic)
	if err != nil || !validDynamic(dynamic) {
		print("FAIL json dynamic unmarshal\n")
		return
	}
	var decodedPayload taggedPayload
	err = json.Unmarshal([]byte(`{"id":9,"display":"decoded","count":"5","values":[3,6],"Ignored":"no","hidden":"no"}`), &decodedPayload)
	if err != nil {
		print("FAIL json struct unmarshal error: " + err.Error() + "\n")
		return
	}
	if decodedPayload.ID != 9 {
		print("FAIL json struct id\n")
		return
	}
	if decodedPayload.Name != "decoded" {
		print("FAIL json struct name\n")
		return
	}
	if decodedPayload.Count != 5 {
		print("FAIL json struct count\n")
		return
	}
	if len(decodedPayload.Values) != 2 || decodedPayload.Values[1] != 6 {
		print("FAIL json struct values\n")
		return
	}
	if decodedPayload.Ignored != "" || decodedPayload.hidden != "" {
		print("FAIL json struct ignored\n")
		return
	}
	writer := &memoryWriter{}
	err = json.NewEncoder(writer).Encode(map[string]string{"ok": "yes"})
	if err != nil || string(writer.data) != "{\"ok\":\"yes\"}\n" {
		print("FAIL json encoder\n")
		return
	}
	reader := &memoryReader{data: []byte("4 5")}
	decoder := json.NewDecoder(reader)
	var first int
	var second int
	if decoder.Decode(&first) != nil || decoder.Decode(&second) != nil || first != 4 || second != 5 {
		print("FAIL json decoder\n")
		return
	}
	var holder rawHolder
	err = json.Unmarshal([]byte(`{"data":{"ok":true}}`), &holder)
	if err != nil || string(holder.Data) != `{"ok":true}` {
		print("FAIL json nested raw\n")
		return
	}
	print("PASS\n")
}

func validDynamic(value any) bool {
	switch object := value.(type) {
	case map[string]any:
		nameValue := object["name"]
		unicodeValue := object["unicode"]
		valuesValue := object["values"]
		name, nameOK := nameValue.(string)
		unicode, unicodeOK := unicodeValue.(string)
		values, valuesOK := valuesValue.([]any)
		return nameOK && name == "renvo" && unicodeOK && unicode == "€" && valuesOK && len(values) == 3
	}
	return false
}
