package render

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	authMW "followup/internal/middleware/auth" // Alias untuk middleware auth
	"followup/internal/models"                 // Untuk GetMenuItems dan MenuKey
)

// TemplateData menyimpan data yang dikirim ke template
type TemplateData struct {
	Title           string
	Active          string
	UserName        string
	Flash           string
	Error           string
	Data            map[string]interface{}
	CSRFToken       string
	IsAuthenticated bool
}

var (
	templateCache     map[string]*template.Template
	templateCacheLock sync.RWMutex
	templateDir       = "./web/templates"
	// Definisikan FuncMap global yang akan kita gunakan
	funcMap = template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"min": func(a, b int) int {
			if a < b {
				return a
			}
			return b
		},
		"isMenuAllowed": func(menuKey models.MenuKey, allowedKeys []models.MenuKey) bool {
			if allowedKeys == nil {
				return false
			}
			for _, k := range allowedKeys {
				if k == menuKey {
					return true
				}
			}
			return false
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
	}
)

// InitTemplates inisialisasi dan cache template
func InitTemplates() error {
	templateCacheLock.Lock()
	defer templateCacheLock.Unlock()

	cache := make(map[string]*template.Template)

	// Get base layout
	baseTemplate := filepath.Join(templateDir, "base.html")

	// Get all pages
	pages, err := filepath.Glob(filepath.Join(templateDir, "pages", "*.html"))
	if err != nil {
		return fmt.Errorf("error getting page templates: %w", err)
	}

	// Get all pages in subdirectories of pages/
	subpages, err := filepath.Glob(filepath.Join(templateDir, "pages", "**", "*.html"))
	if err != nil {
		return fmt.Errorf("error getting subpage templates: %w", err)
	}
	pages = append(pages, subpages...)

	// Get all partials
	partials, err := filepath.Glob(filepath.Join(templateDir, "partials", "*.html"))
	if err != nil {
		return fmt.Errorf("error getting partial templates: %w", err)
	}
	// Get all partials in subdirectories of partials/ (jika ada)
	subpartials, err := filepath.Glob(filepath.Join(templateDir, "partials", "**", "*.html"))
	if err != nil {
		return fmt.Errorf("error getting sub-partial templates: %w", err)
	}
	partials = append(partials, subpartials...)

	// Create template for each page
	for _, page := range pages {
		name := filepath.Base(page)

		// For subdirectory pages, include the subdirectory in the name relative to 'pages'
		// Example: pages/upload/excel.html -> upload/excel.html
		relPath, _ := filepath.Rel(filepath.Join(templateDir, "pages"), page)
		relPath = filepath.ToSlash(relPath) // Normalize path separator
		if strings.Contains(relPath, "/") {
			name = relPath
		}

		// Files to parse for this page (page itself + all partials)
		filesToParse := []string{page}
		filesToParse = append(filesToParse, partials...)

		// Parse page without base template (standalone, e.g., for HTMX fragments not needing full layout)
		// The name of the template will be 'name' (e.g., "user.html" or "upload/excel.html")
		// All partials will be associated with this template under their respective names (e.g., "partials/user_form.html")
		ts, err := template.New(name).Funcs(funcMap).ParseFiles(filesToParse...)
		if err != nil {
			return fmt.Errorf("error parsing page template %s with partials: %w. Files: %v", page, err, filesToParse)
		}
		cache[name] = ts

		// Files to parse for this page with base (page + base + all partials)
		filesToParseWithBase := []string{baseTemplate, page}
		filesToParseWithBase = append(filesToParseWithBase, partials...)

		// Create version with base template
		// The name of the main template here is baseTemplate's base name (e.g., "base.html")
		// It will define templates for "base.html", the 'page' (e.g., "user.html"), and all partials.
		tsBase, err := template.New(filepath.Base(baseTemplate)).Funcs(funcMap).ParseFiles(filesToParseWithBase...)
		if err != nil {
			return fmt.Errorf("error parsing page %s with base template %s and partials: %w. Files: %v", page, baseTemplate, err, filesToParseWithBase)
		}
		// Add to cache with "base:" prefix (e.g., "base:user.html")
		cache["base:"+name] = tsBase
	}

	// Parse standalone templates (e.g., login.html, 404.html in web/templates/)
	// These might also want to use partials, so we include partials here too.
	standaloneTemplates, err := filepath.Glob(filepath.Join(templateDir, "*.html"))
	if err != nil {
		return fmt.Errorf("error getting standalone templates: %w", err)
	}

	for _, tmplPath := range standaloneTemplates {
		// Skip base.html as it's not a standalone page to be rendered directly by this logic
		if filepath.Base(tmplPath) == filepath.Base(baseTemplate) {
			continue
		}

		name := filepath.Base(tmplPath)
		filesToParse := []string{tmplPath}
		filesToParse = append(filesToParse, partials...)

		ts, err := template.New(name).Funcs(funcMap).ParseFiles(filesToParse...)
		if err != nil {
			return fmt.Errorf("error parsing standalone template %s with partials: %w. Files: %v", tmplPath, err, filesToParse)
		}
		cache[name] = ts
	}

	// Re-parse all partials individually so they can be called directly if needed,
	// for example, if a handler wants to return ONLY a partial.
	// Their names in the cache will be their path relative to templateDir (e.g., "partials/user_form.html")
	for _, partialPath := range partials {
		// Name for cache: relative path from templateDir, e.g., "partials/user_form.html"
		partialName, err := filepath.Rel(templateDir, partialPath)
		if err != nil {
			return fmt.Errorf("error getting relative path for partial %s: %w", partialPath, err)
		}
		partialName = filepath.ToSlash(partialName) // Normalize to use forward slashes

		// Each partial is parsed as its own template set, also knowing about other partials (though less common to cross-call)
		// For simplicity here, we parse each partial by itself. If partials call other partials,
		// they would need to be parsed together like pages.
		// However, the primary goal is that `{{ template "partials/name.html" . }}` works from a page.
		// The page's template set (which includes all partials) will handle this.
		// This loop is more for if you ever call render.Template(w, r, "partials/user_form.html", data)

		// Parse the partial file. The template name given to New() will be the one it's accessible by if we parse only one file.
		// To ensure it's stored as "partials/user_form.html", we use that name for New().
		tsPartial, err := template.New(partialName).Funcs(funcMap).ParseFiles(partialPath)
		if err != nil {
			return fmt.Errorf("error parsing individual partial template %s as %s: %w", partialPath, partialName, err)
		}
		cache[partialName] = tsPartial
	}

	// Assign to global cache
	templateCache = cache
	return nil
}

// AddDefaultData menambahkan data default ke template
func AddDefaultData(td *TemplateData, r *http.Request) *TemplateData {
	if td == nil {
		td = &TemplateData{Data: make(map[string]interface{})} // Inisialisasi Data map
	} else if td.Data == nil {
		td.Data = make(map[string]interface{}) // Pastikan Data map diinisialisasi
	}

	// Ambil AllowedMenuKeys dari context
	if r != nil && r.Context() != nil {
		if allowedMenusCtx := r.Context().Value(authMW.AllowedMenuKeysKey); allowedMenusCtx != nil {
			if allowedMenus, ok := allowedMenusCtx.([]models.MenuKey); ok {
				td.Data["AllowedMenuKeys"] = allowedMenus
			} else {
				log.Println("[RENDER] Peringatan: AllowedMenuKeysKey ditemukan di konteks tetapi bukan tipe []models.MenuKey")
				td.Data["AllowedMenuKeys"] = []models.MenuKey{} // Default jika tipe salah
			}
		} else {
			// log.Println("[RENDER] AllowedMenuKeysKey tidak ditemukan di konteks.")
			td.Data["AllowedMenuKeys"] = []models.MenuKey{} // Default ke slice kosong jika tidak ditemukan (misalnya, path publik)
		}
	}

	// Tambahkan semua item menu yang tersedia
	td.Data["AllMenuItems"] = models.GetMenuItems()

	return td
}

// Template renders a template without base layout
func Template(w http.ResponseWriter, r *http.Request, tmpl string, data interface{}) {
	// Convert data to TemplateData if needed
	var td *TemplateData
	if data != nil {
		if tData, ok := data.(*TemplateData); ok {
			td = tData
		} else if mapData, ok := data.(map[string]interface{}); ok {
			td = &TemplateData{
				Data: mapData,
			}

			// Extract common fields
			if title, ok := mapData["Title"].(string); ok {
				td.Title = title
			}
			if active, ok := mapData["Active"].(string); ok {
				td.Active = active
			}
			if userName, ok := mapData["UserName"].(string); ok {
				td.UserName = userName
			}
		}
	}

	// Add default data
	td = AddDefaultData(td, r)

	// Render template
	renderTemplate(w, tmpl, td)
}

// TemplateWithBase renders a template with base layout
func TemplateWithBase(w http.ResponseWriter, r *http.Request, tmpl string, data interface{}) {
	// Convert data to TemplateData
	var td *TemplateData
	if data != nil {
		if tData, ok := data.(*TemplateData); ok {
			td = tData
		} else if mapData, ok := data.(map[string]interface{}); ok {
			td = &TemplateData{
				Data: mapData,
			}

			// Extract common fields
			if title, ok := mapData["Title"].(string); ok {
				td.Title = title
			}
			if active, ok := mapData["Active"].(string); ok {
				td.Active = active
			}
			if userName, ok := mapData["UserName"].(string); ok {
				td.UserName = userName
			}
		}
	}

	// Add default data
	td = AddDefaultData(td, r)

	// Render template with base
	renderTemplate(w, "base:"+tmpl, td)
}

// renderTemplate renders a template to the response writer
func renderTemplate(w http.ResponseWriter, tmplWithOptionalFragment string, data *TemplateData) {
	cacheKey := tmplWithOptionalFragment
	templateToExecuteName := ""

	if strings.Contains(tmplWithOptionalFragment, "#") {
		parts := strings.SplitN(tmplWithOptionalFragment, "#", 2)
		cacheKey = parts[0]              // e.g., "user.html"
		templateToExecuteName = parts[1] // e.g., "user_form_content"
	}

	// Get template from cache
	templateCacheLock.RLock()
	t, ok := templateCache[cacheKey]
	templateCacheLock.RUnlock()

	if !ok {
		// Template not found in cache
		log.Printf("[RENDER] Template set with cache key '%s' not found in cache", cacheKey)
		http.Error(w, "Template set not found: "+cacheKey, http.StatusInternalServerError)
		return
	}

	// Create buffer to render template
	buf := new(bytes.Buffer)
	var err error

	if templateToExecuteName != "" {
		log.Printf("[RENDER] Rendering specific template '%s' from cache key '%s'. Base template object: %v", templateToExecuteName, cacheKey, t.Name())
		err = t.ExecuteTemplate(buf, templateToExecuteName, data)
	} else {
		log.Printf("[RENDER] Rendering main template from cache key '%s'. Base template object: %v", cacheKey, t.Name())
		err = t.Execute(buf, data) // Execute the default template in the set (usually t.Name() or the first one parsed)
	}

	if err != nil {
		log.Printf("[RENDER] Error executing template (key: %s, name: '%s'): %v", cacheKey, templateToExecuteName, err)
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}

	log.Printf("[RENDER] Buffer length for %s (executing '%s'): %d", cacheKey, templateToExecuteName, buf.Len())
	// Untuk debugging, jangan lakukan ini di produksi karena bisa sangat verbose:
	if buf.Len() < 500 && buf.Len() > 0 { // Log isi buffer jika kecil (tapi tidak kosong)
		log.Printf("[RENDER] Buffer content for %s (executing '%s', first 500 bytes): %s", cacheKey, templateToExecuteName, buf.String())
	} else if buf.Len() == 0 {
		log.Printf("[RENDER] Buffer for %s (executing '%s') is EMPTY. This might be intended or an issue if content was expected.", cacheKey, templateToExecuteName)
	}

	// Write rendered template to response
	_, err = buf.WriteTo(w)
	if err != nil {
		log.Printf("[RENDER] Error writing template %s to response: %v", cacheKey, err)
		http.Error(w, "Error writing response", http.StatusInternalServerError)
	}
}
