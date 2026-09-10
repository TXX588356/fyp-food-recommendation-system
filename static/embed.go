package static

import "embed"

// FS contains the compiled React frontend for serverless deployments.
//
//go:embed index.html assets/* dish.png favicon.svg icons.svg malaysia_cities.csv
var FS embed.FS
