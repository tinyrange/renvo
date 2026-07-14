package main

type T struct { f struct{ A int } }
func f(v struct{ A int }) struct{ A int } { return v }
type Alias = struct{ A int }
type Rows []struct{ A int }
func main() { var x struct{ A int }; y := struct{ A int }{1}; var z struct{ A int } = struct{ A int }{2}; a := Alias{6}; xs := []struct{ A int }{{3}}; rows := Rows{{4}}; t := T{f: struct{ A int }{5}}; _ = f(x).A + y.A + z.A + a.A + xs[0].A + rows[0].A + t.f.A }
