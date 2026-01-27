//go:build embed_frontend

package frontend

import (
	"embed"
	"io/fs"
)

//go:embed beitraege/*
var beitraegeEmbed embed.FS

func init() {
	var err error
	BeitraegeFS, err = fs.Sub(beitraegeEmbed, "beitraege")
	if err != nil {
		panic("failed to create beitraege sub-filesystem: " + err.Error())
	}
}
