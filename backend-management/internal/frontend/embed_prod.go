//go:build embed_frontend

package frontend

import (
	"embed"
	"io/fs"
)

//go:embed plan/*
var planEmbed embed.FS

//go:embed zeit/*
var zeitEmbed embed.FS

func init() {
	var err error
	PlanFS, err = fs.Sub(planEmbed, "plan")
	if err != nil {
		panic("failed to create plan sub-filesystem: " + err.Error())
	}
	ZeitFS, err = fs.Sub(zeitEmbed, "zeit")
	if err != nil {
		panic("failed to create zeit sub-filesystem: " + err.Error())
	}
}
