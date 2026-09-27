package integrations

func init() {
	Register(arrKind{
		kind:        "radarr",
		name:        "Radarr",
		port:        7878,
		libraryPath: "/api/v3/movie",
		library:     KPI{Key: "movies", Label: "Movies"},
	})
}
