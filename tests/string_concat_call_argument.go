package main

func rtgStringConcatCallArgValue(s string) string {
	return s
}

func rtgStringConcatCallArgJoin(a string, b string) string {
	return rtgStringConcatCallArgValue(a + "/" + b)
}

func appMain(args []string) int {
	if rtgStringConcatCallArgJoin("PASS", "OK") == "PASS/OK" {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
