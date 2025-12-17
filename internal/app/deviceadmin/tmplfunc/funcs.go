// Package tmplfunc contains extension functions for templates
package tmplfunc

import (
	"html/template"
	"net/url"
)

type TurboStreamSigner func(streamName string) (hash string)

func FuncMap(h HashedNamers, tss TurboStreamSigner) template.FuncMap {
	return template.FuncMap{
		"queryEscape":     url.QueryEscape,
		"appHashed":       h.AppHashed,
		"staticHashed":    h.StaticHashed,
		"isIPAddr":        IsIPAddr,
		"signTurboStream": tss,
	}
}
