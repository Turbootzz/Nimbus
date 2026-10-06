package integrations

func init() {
	Register(arrKind{
		kind:        "sonarr",
		name:        "Sonarr",
		port:        8989,
		libraryPath: "/api/v3/series",
		library:     KPI{Key: "series", Label: "Series"},
	})
}
