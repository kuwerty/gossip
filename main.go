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
    "os/exec"
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


//
// Scope is a nested map of keys to values used as the context in template
// execution. Templates can use the {{ .Value "foo" }} notation to search
// parent scopes or just {{ .foo }} to search the local scope.
//
type KeyVals map[string]string

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


//
// Generator is the main 
//
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

func (g *Generator) PushScope(values KeyVals) {
    g.Scope = &Scope{
        values : values,
        parent : g.Scope,
    }
}

func (g *Generator) PopScope() {
    if g.Scope != nil {
        g.Scope = g.Scope.parent
    }
}

func (g *Generator) ParseArgs(args []string) KeyVals {

    res := make(KeyVals)

    for _, arg := range(args) {
        kv := strings.SplitN(arg, "=", 2)
        if len(kv) != 2 {
            abort("bad argument %s\n", arg)
        } else {
            res[kv[0]] = kv[1]
        }
    }

    return res
}

func (g *Generator) macro(name string, args ...string) string {
    printf("macro %s %s\n", name, strings.Join(args, ","))

    // create new scope
    g.PushScope(g.ParseArgs(args))
    defer g.PopScope()

    // look up macro name
    buf := bytes.NewBuffer(nil)
    err := g.Master.ExecuteTemplate(buf, name, g.Scope.values)
    if err != nil {
        abort("%s", err)
    }

    return buf.String()
}

func (g *Generator) markdown(filename string, args ...string) string {
    // read mark down file
    contents, err := ioutil.ReadFile(filename)
    if err != nil {
        abort( "error reading %s: %s\n", filename, err )
    }

    // parse markdown using template language
    t := g.Parse(string(contents))

    // create new scope and execute the template
    g.PushScope(g.ParseArgs(args))
    defer g.PopScope()

    buf := bytes.NewBuffer(nil)
    err = t.Execute(buf, nil)
    if err != nil {
        abort("ff%s", err)
    }

    // convert markdown to html
    basename  := path.Base(filename)
    extension := path.Ext(basename)
    name      := basename[0:len(basename)-len(extension)]

    htmlFlags := blackfriday.HTML_TOC |
                 blackfriday.HTML_GITHUB_BLOCKCODE;

    renderer := blackfriday.HtmlRenderer(htmlFlags, name, "")

    html := blackfriday.Markdown(buf.Bytes(), renderer, blackfriday.EXTENSION_AUTOLINK)

    return string(html)
}



func main() {
    var flgServer = flag.Bool("server", false, "start http server after compilation")

    flag.Parse()

    for _,arg := range flag.Args() {
        g := new(Generator)

        g.Funcs = template.FuncMap {
            "macro"     : g.macro,
            "markdown"  : g.markdown,
        }

        g.Master = template.New("master").Funcs(g.Funcs)


        dir := path.Dir(arg)
        name := path.Base(arg)

        printf("directory %s\n", arg)

        err := os.Chdir(dir)
        if err != nil {
            abort("%s", err)
        }

        // parse all the templates we can find.
        filepath.Walk(".", func(filename string, info os.FileInfo, err error) error {

            if info != nil && !info.IsDir() {
                printf("  template:%s\n", filename)

                g.Compile(filename)
            }

            return nil
        })

        printf("  executing:%s\n", name)

        err = g.Master.ExecuteTemplate(os.Stdout, name, nil)
        if err != nil {
            abort("%s", err)
        }
    }

    //
    // Start server if requested
    //
    if *flgServer {
        go func() {
            http.Handle("/", http.FileServer(http.Dir("../public")))

            printf("listening on http://127.0.0.1:5000/\n")

            log.Fatal(http.ListenAndServe(":5000", nil))
        }()

        // start a browser. there is a race condition but the chances
        // of chrome or safari starting up before the server is tiny.
        exec.Command("open", "http://127.0.0.1:5000/").Run()

        // sleep forever
        select{}
    }
}
