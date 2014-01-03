package main

import (
    "text/template"
    "flag"
    "fmt"
    "log"
    "os"
    "io/ioutil"
    "bytes"
    "path"
    "strings"
    "path/filepath"
    "github.com/russross/blackfriday"
    "net/http"
)

func printf(format string, args ...interface{}) {
    fmt.Fprintf(os.Stderr, format, args...)
}


func abort(format string, args ...interface{}) {
    fmt.Fprintf(os.Stderr, format, args...)
    fmt.Fprintln(os.Stderr, "")
    os.Exit(1)
}

type Scope struct {
    parent  *Scope
    values  map[string]string
}

func (s *Scope) Value(name string) interface{} {
    value, present := s.values[name]
    if present {
        return value
    }

    if s.parent != nil {
        return s.parent.Value(name)
    }

    return nil
}

type Generator struct {
    Master  *template.Template
    Funcs   template.FuncMap
    Scope   *Scope
}

func (g *Generator) Compile(filename string) *template.Template {
    t := g.Master.New(filename).Funcs(g.Funcs)

    return template.Must(t.ParseFiles(filename))
}

func (g *Generator) Parse(contents string) *template.Template {
    t := g.Master.New("").Funcs(g.Funcs)

    return template.Must(t.Parse(contents))
}

func (g *Generator) PushScope() {
    scope := &Scope{
        values : make(map[string]string),
    }
    scope.parent = g.Scope
    g.Scope = scope
}

func (g *Generator) PopScope() {
    if g.Scope != nil {
        g.Scope = g.Scope.parent
    }
}


func (g *Generator) macro(name string, args ...string) string {
    printf("macro %s %s\n", name, strings.Join(args, ","))

    // create new scope
    g.PushScope()
    defer g.PopScope()

    // parse args
    for _, arg := range(args) {
        kv := strings.SplitN(arg, "=", 2)
        if len(kv) != 2 {
            abort("bad argument %s\n", arg)
        } else {
            g.Scope.values[kv[0]] = kv[1]
        }
    }

    // look up macro name
    buf := bytes.NewBuffer(nil)
    err := g.Master.ExecuteTemplate(buf, name, g.Scope.values)
    if err != nil {
        abort("%s", err)
    }

    return buf.String()
}

func (g *Generator) markdown(filename string) string {
    basename  := path.Base(filename)
    extension := path.Ext(basename)
    name      := basename[0:len(basename)-len(extension)]

    htmlFlags := blackfriday.HTML_TOC |
                 blackfriday.HTML_GITHUB_BLOCKCODE;

    renderer := blackfriday.HtmlRenderer(htmlFlags, name, "")

    contents, err := ioutil.ReadFile(filename)
    if err != nil {
        abort( "error reading %s: %s\n", filename, err )
    }

    t := g.Parse(string(contents))

    buf := bytes.NewBuffer(nil)

    err = t.Execute(buf, nil)
    if err != nil {
        abort("ff%s", err)
    }

    html := blackfriday.Markdown(buf.Bytes(), renderer, blackfriday.EXTENSION_AUTOLINK)

    return string(html)
}

func main() {
    var flgIndex = flag.String("index", "index.html", "specify index file to render")

    flag.Parse()

    if flgIndex == nil {
        abort("must specify an index")
    }

    index := *flgIndex;


    g := new(Generator)

    g.Funcs = template.FuncMap {
        "macro"     : g.macro,
        "markdown"  : g.markdown,
    }

    g.Master = template.New("master").Funcs(g.Funcs)

    walk := func(filename string, info os.FileInfo, err error) error {

        if info != nil && !info.IsDir() {
            printf("template:%s\n", filename)

            g.Compile(filename)
        }

        return nil
    }

    for _,arg := range flag.Args() {
        dir := path.Dir(arg)

        os.Chdir(dir)

        filepath.Walk(".", walk)
    }

    err := g.Master.ExecuteTemplate(os.Stdout, index, nil)
    if err != nil {
        abort("%s", err)
    }

    http.Handle("/", http.FileServer(http.Dir("../public")))

    printf("listening on http://127.0.0.1:5000/")

    log.Fatal(http.ListenAndServe(":5000", nil))
}
