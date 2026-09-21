package vendor

// Manifest pins every downloaded vendored file: exact version, CDN-relative
// path, sha256. The pin is the compat contract — companion scripts and glue
// in the binary are authored for exactly these versions. To bump a library:
// update the version in the paths below and the sha256 of the fresh bytes;
// the next `pharos start` refetches exactly what changed.

//nolint:lll // sha256 hex lines are long by nature
func init() {
	Manifest = []Entry{
		// mermaid 11.x — primary lib for .mermaid diagrams.
		{Lib: "mermaid", Version: "11.17.2", File: "mermaid.min.js",
			Path:   "npm/mermaid@11.17.2/dist/mermaid.min.js",
			SHA256: "581ed7d74bd9048d0e3a91363927d72ef22942d7722546b27f7cc29e35390eb8"},

		// sql-workbench v0.3.0 — single-file web component (LEARN-194),
		// released from udit-001/sql-workbench.
		{Lib: "sql-workbench", Version: "v0.3.0", File: "sql-workbench.js",
			Path:   "gh/udit-001/sql-workbench@v0.3.0/dist/sql-workbench.js",
			SHA256: "520ab78be3e41993c1a3c5a84a0bd258dd11fa927587983f182fbf0fd43ab278"},

		// highlight.js 11.11.1 — common build (language-* opt-in).
		{Lib: "highlightjs", Version: "11.11.1", File: "highlight.min.js",
			Path:   "gh/highlightjs/cdn-release@11.11.1/build/highlight.min.js",
			SHA256: "c4a399dd6f488bc97a3546e3476747b3e714c99c57b9473154c6fb8d259b9381"},

		// KaTeX 0.16.22 — js, css, auto-render contrib, and the 20 woff2
		// fonts its CSS references (relative url(fonts/…), fully local).
		{Lib: "katex", Version: "0.16.22", File: "katex.min.js",
			Path:   "npm/katex@0.16.22/dist/katex.min.js",
			SHA256: "e8d885505949f3a5f4abdd5dd0d53696bd1371ad26ffbf4f310dcd77c8cdae89"},
		{Lib: "katex", Version: "0.16.22", File: "katex.min.css",
			Path:   "npm/katex@0.16.22/dist/katex.min.css",
			SHA256: "19095127357ed6d29fe0a63a6b000c913a89f7f1963b765dd3715e97c9852e75"},
		{Lib: "katex", Version: "0.16.22", File: "contrib/auto-render.min.js",
			Path:   "npm/katex@0.16.22/dist/contrib/auto-render.min.js",
			SHA256: "bb53eb953394531aae36fdd537065c4244eb8542901a3ce914601d932675b8ac"},
		katexFont("KaTeX_AMS-Regular", "0cdd387c9590a1a9f9794560022dbb59654a7d86f187aa0c81495ad42d3a7308"),
		katexFont("KaTeX_Caligraphic-Bold", "de7701e42cf1f4cf0b766c03fb27977207eee2f4fd5d76fa82188406da43ea4c"),
		katexFont("KaTeX_Caligraphic-Regular", "5d53e70ad607c2352162dec9e0923fb54ecdafaccbf604cd8dcf7d00facb989b"),
		katexFont("KaTeX_Fraktur-Bold", "74444efd593c005e3f4573b44524704c0af0a937fe911cca9e94068d0d140d3f"),
		katexFont("KaTeX_Fraktur-Regular", "51814d270d06ff0255dba0799994fa4d8c84d11f09951d47595f4abb1f3602dc"),
		katexFont("KaTeX_Main-Bold", "0f60d1b897938ec918c8ce073092411baf9438f6739465693ff18b0f9d20b021"),
		katexFont("KaTeX_Main-BoldItalic", "99cd42a3c072d918f2f44984a807cf7aa16e13545fd0875fc07c6c65f99e715b"),
		katexFont("KaTeX_Main-Italic", "97479ca6cce906abc961ecac96faa5f9ca2e61b8e7670d475826bcdee9a7c267"),
		katexFont("KaTeX_Main-Regular", "c2342cd8b869e01752a9321dc17213fc40d4d04c79688c1d43f2cf316abd7866"),
		katexFont("KaTeX_Math-BoldItalic", "dc47344dbb6cb5b655c8460d561f4df5f501b90c804ad3c6cec65fe322351ab1"),
		katexFont("KaTeX_Math-Italic", "7af58c5ec8f132a2ddde9027c6d7814decce4d3b822a11192a42a20e2e973264"),
		katexFont("KaTeX_SansSerif-Bold", "e99ae51144bf1232efcc1bfe5add36262c6866b0faab24fa75740e1b98577a62"),
		katexFont("KaTeX_SansSerif-Italic", "00b26ac825e2095056396e0553b8ac26d3f8ad158c3826e28b4c45b385c4714a"),
		katexFont("KaTeX_SansSerif-Regular", "68e8c73ef42afd3ccec58bf0fba302cce448938e7fc020a5e31f8a952eee1342"),
		katexFont("KaTeX_Script-Regular", "036d4e95149b69ff9bcc0cd55771efeb25ffa3947293e69acd78d5ac328c684b"),
		katexFont("KaTeX_Size1-Regular", "6b47c40166b6dbe21a5dfca7718413f2147fd2399be1ba605d8ad39cedf25dfe"),
		katexFont("KaTeX_Size2-Regular", "d04c54219f9eaec6d4d4fd42dfb28785975a4794d6b2fc71e566b9cd6db842dd"),
		katexFont("KaTeX_Size3-Regular", "73d591271b1604960cb10bb90fee021670af7297017e0e98480b332d11f51995"),
		katexFont("KaTeX_Size4-Regular", "a4af7d414440a1c1790825cfb700cf9cf43b0f2c4b04f0ebc523011ad9853ec0"),
		katexFont("KaTeX_Typewriter-Regular", "71d517d67827787cfabdf186914cc3358eda539e37931941f2b2fd4a21f68c0b"),

		// Vega stack — vega-lite is the primary; vega and vega-embed ship
		// different majors and are pinned separately (compat contract). Each
		// co-lib keeps its own version label: the cache key is file-level.
		{Lib: "vega", Version: "6.4.3", File: "vega-lite.min.js",
			Path:   "npm/vega-lite@6.4.3/build/vega-lite.min.js",
			SHA256: "35a9821df838825b05a6a73e9414b58747a1b18321583858ed903c66393a5c7e"},
		{Lib: "vega", Version: "6.4.0", File: "vega.min.js",
			Path:   "npm/vega@6.4.0/build/vega.min.js",
			SHA256: "8f6a3587cf8d4f42c7e08120e3eb05d067e746d554e39d2dcf52acc0bd5ba28f"},
		{Lib: "vega", Version: "7.2.0", File: "vega-embed.min.js",
			Path:   "npm/vega-embed@7.2.0/build/vega-embed.min.js",
			SHA256: "b69eac2846a0061683b7e03501790fb0bcbdb851c797c6baf3417c9d8852819e"},
	}
}

// katexFont builds the manifest entry for one KaTeX woff2 font.
func katexFont(name, sha string) Entry {
	return Entry{
		Lib:     "katex",
		Version: "0.16.22",
		File:    "fonts/" + name + ".woff2",
		Path:    "npm/katex@0.16.22/dist/fonts/" + name + ".woff2",
		SHA256:  sha,
	}
}
