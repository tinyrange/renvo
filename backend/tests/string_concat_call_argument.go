package main

func renvoStringConcatCallArgValue(s string) string {
	return s
}

func renvoStringConcatCallArgJoin(a string, b string) string {
	return renvoStringConcatCallArgValue(a + "/" + b)
}

func appMain(args []string) int {
	if renvoStringConcatCallArgJoin("PASS", "OK") == "PASS/OK" {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
