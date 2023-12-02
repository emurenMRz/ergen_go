package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"slices"
	"strings"

	"github.com/emurenMRz/ergen_go/cmd/pg_ergen/internal/db"
)

var pageTemplate = `
<div style="display:flex;">
	<div style="padding: 0 1rem 0 0">%s</div>
	<div id="svg"></div>
</div>
<script>
	function onClick(s) {
		fetch(s)
			.then(r => r.text())
			.then(svg => {
				const e = document
				.getElementById("svg");
				e.textContent = ""

				const btn = document.createElement("button");
				btn.innerHTML = "Download";
				btn.onclick = () => {
					const svgNode = document.getElementById("svg-node");
					const svgText = new XMLSerializer().serializeToString(svgNode);
					const svgBlob = new Blob([svgText], { type: 'image/svg+xml' });
					const svgUrl = URL.createObjectURL(svgBlob);
				  
					const a = document.createElement("a");
					a.href = svgUrl;
					a.download = "ER " + s + ".svg";
					a.click();

					URL.revokeObjectURL(svgUrl);
				};
				e.appendChild(btn);

				const img = document.createElement("div");
				img.id = "svg-node";
				img.innerHTML = svg;
				e.appendChild(img);
			})
			.catch(console.error);
	}
</script>
`

func server(conn db.DBConnect, names []string, acceptPort uint16) {
	indexPage := "<ul>"
	for i := range names {
		indexPage += fmt.Sprintf(`<li><a href="#" onclick="javascript:onClick('%s')">%s</a></li>`, names[i], names[i])
	}
	indexPage += "<ul>"

	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, pageTemplate, indexPage)
			return
		}

		if strings.HasPrefix(path, "/") {
			filename := path[1:]
			if slices.Contains(names, filename) {
				w.Header().Set("Content-Type", "image/svg+xml")
				c := connectDatabase(conn, filename)
				c.OutputSVG(w)
				return
			}
		}
		http.NotFound(w, r)
	}))

	url := fmt.Sprintf("http://localhost:%d/", acceptPort)
	open(url)
	log.Println("Connection: " + url)
	err := http.ListenAndServe(fmt.Sprintf(":%d", acceptPort), nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func open(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default:
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}
