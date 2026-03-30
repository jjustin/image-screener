package main

import (
	"html/template"
	"net/http"
	"sort"
	"strings"
)

var adminTmpl = template.Must(template.New("admin").Parse(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Admin</title>
  <style>
    * { box-sizing: border-box; }
    body { font-family: sans-serif; margin: 2rem; background: #f5f5f5; color: #222; }
    h1 { margin-bottom: 0.25rem; }
    h2 { margin-top: 2rem; margin-bottom: 0.75rem; font-size: 1.1rem; }
    .count { font-weight: normal; color: #888; }
    .grid { display: flex; flex-wrap: wrap; gap: 0.75rem; }
    .item { position: relative; background: #fff; border-radius: 6px; overflow: hidden; box-shadow: 0 1px 3px rgba(0,0,0,.15); }
    .item img { width: 160px; height: 120px; object-fit: cover; display: block; }
    .del { position: absolute; top: 4px; right: 4px; background: rgba(180,0,0,0.85); color: #fff; border: none; border-radius: 4px; cursor: pointer; padding: 2px 7px; font-size: 0.85rem; line-height: 1.4; }
    .del:hover { background: #cc0000; }
    .empty { color: #999; font-style: italic; margin: 0; }
  </style>
</head>
<body>
  <h1>Admin Panel</h1>
  {{range .Screens}}
  <h2>Screen {{.ID}} <span class="count">({{len .Images}} image{{if ne (len .Images) 1}}s{{end}})</span></h2>
  {{if .Images}}
  <div class="grid">
    {{range .Images}}
    <div class="item" id="img-{{.ScreenID}}-{{.Filename}}">
      <img src="/api/admin/image/{{.ScreenID}}/{{.Filename}}" alt="{{.Filename}}">
      <button class="del" data-screen="{{.ScreenID}}" data-file="{{.Filename}}">&#x2715;</button>
    </div>
    {{end}}
  </div>
  {{else}}
  <p class="empty">No images uploaded yet.</p>
  {{end}}
  {{end}}
  <script>
    document.addEventListener('click', async function(e) {
      const btn = e.target.closest('.del');
      if (!btn) return;
      if (!confirm('Delete this image?')) return;
      const screen = btn.dataset.screen;
      const file = btn.dataset.file;
      const res = await fetch('/api/admin/delete/' + screen + '/' + file, {method: 'POST'});
      if (res.ok) {
        document.getElementById('img-' + screen + '-' + file).remove();
      } else {
        alert('Delete failed (' + res.status + ')');
      }
    });
  </script>
</body>
</html>`))

func basicAuth(password string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, pass, ok := r.BasicAuth()
		if !ok || pass != password {
			w.Header().Set("WWW-Authenticate", `Basic realm="Admin"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (h *Handlers) AdminPanel(w http.ResponseWriter, r *http.Request) {
	allImages := h.store.AllImages()
	screens := h.store.Screens()
	sort.Strings(screens)

	type imageData struct {
		ScreenID string
		Filename string
	}
	type screenData struct {
		ID     string
		Images []imageData
	}

	data := make([]screenData, 0, len(screens))
	for _, id := range screens {
		var imgs []imageData
		for _, fn := range allImages[id] {
			imgs = append(imgs, imageData{ScreenID: id, Filename: fn})
		}
		data = append(data, screenData{ID: id, Images: imgs})
	}
	adminTmpl.Execute(w, map[string]any{"Screens": data})
}

func (h *Handlers) AdminImage(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/api/admin/image/"), "/", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	path, ok := h.store.GetImagePath(parts[0], parts[1])
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, path)
}

func (h *Handlers) AdminDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/api/admin/delete/"), "/", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	screenID, filename := parts[0], parts[1]
	if !h.store.HasScreen(screenID) {
		http.NotFound(w, r)
		return
	}
	if err := h.store.Delete(screenID, filename); err != nil {
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
