package integrations

func init() {
	Register(arrKind{
		kind:        "radarr",
		name:        "Radarr",
		port:        7878,
		libraryPath: "/api/v3/movie",
		library:     KPI{Key: "movies", Label: "Movies"},
		// Radarr 3 and 4 have no /wanted/missing
		wantedFromLibrary: true,
	})
}
