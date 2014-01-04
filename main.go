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
    "strconv"
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

// relaxed bool parsing, defaults to false on error
func str2bool(str string) bool {
    val, err := strconv.ParseBool(str)
    if err != nil {
        return false
    }
    return val
}

// split string on equals sign and return both parts
func spliteq(arg string) (string, string) {
    kv := strings.SplitN(arg, "=", 2)
    if len(kv) != 2 {
        abort("bad argument %s\n", arg)
    }
    return kv[0], kv[1]
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
// Generator is the gossip execution engine
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
        k, v := spliteq(arg)
        res[k] = v
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

func (g *Generator) markdown(filename string, argsv ...string) string {
    // read mark down file
    contents, err := ioutil.ReadFile(filename)
    if err != nil {
        abort( "error reading %s: %s\n", filename, err )
    }

    // parse markdown using template language
    t := g.Parse(string(contents))

    // create new scope and execute the template
    args := g.ParseArgs(argsv)
    g.PushScope(args)
    defer g.PopScope()

    buf := bytes.NewBuffer(nil)
    err = t.Execute(buf, nil)
    if err != nil {
        abort("ff%s", err)
    }

    // set up the HTML renderer
    htmlFlags := 0
    htmlFlags |= blackfriday.HTML_USE_XHTML
    htmlFlags |= blackfriday.HTML_USE_SMARTYPANTS
    htmlFlags |= blackfriday.HTML_SMARTYPANTS_FRACTIONS
    htmlFlags |= blackfriday.HTML_SMARTYPANTS_LATEX_DASHES
    htmlFlags |= blackfriday.HTML_SKIP_SCRIPT
    htmlFlags |= blackfriday.HTML_GITHUB_BLOCKCODE

    // set up the parser
    extensions := 0
    extensions |= blackfriday.EXTENSION_NO_INTRA_EMPHASIS
    extensions |= blackfriday.EXTENSION_TABLES
    extensions |= blackfriday.EXTENSION_FENCED_CODE
    extensions |= blackfriday.EXTENSION_AUTOLINK
    extensions |= blackfriday.EXTENSION_STRIKETHROUGH
    extensions |= blackfriday.EXTENSION_SPACE_HEADERS
    extensions |= blackfriday.EXTENSION_FOOTNOTES;

    if str2bool(args["toc"]) {
        htmlFlags |= blackfriday.HTML_TOC;
    }

    // convert markdown to html
    basename  := path.Base(filename)
    extension := path.Ext(basename)
    name      := basename[0:len(basename)-len(extension)]

    renderer := blackfriday.HtmlRenderer(htmlFlags, name, "")

    html := blackfriday.Markdown(buf.Bytes(), renderer, extensions)

    return string(html)
}



// Need a type to store multiple string arguments
type stringslice []string
func (s *stringslice) String() string { return fmt.Sprintf("%d", *s) }
func (s *stringslice) Set(value string) error { *s = append(*s, value); return nil }



func main() {
    defines := make(stringslice, 0)

    output := ""

    var flgServer = flag.Bool("server", false, "start http server after compilation.")

    flag.StringVar(&output, "output", "", "output file.")
    flag.StringVar(&output, "o",      "", "output file.")

    flag.Var(&defines, "D", "define values for the initial context")

    flag.Parse()

    if output == "" {
        flag.PrintDefaults()

        abort("must specify output file")
    }

    outputWriter, err := os.Create(output)
    if err != nil {
        abort("cannot open output: %s", err)
    }

    for _,arg := range flag.Args() {
        g := new(Generator)

        g.Funcs = template.FuncMap {
            "macro"     : g.macro,
            "markdown"  : g.markdown,
        }

        g.Master = template.New("master").Funcs(g.Funcs)

        g.PushScope(g.ParseArgs(defines))
        defer g.PopScope()

        dir := path.Dir(arg)
        name := path.Base(arg)

        printf("directory %s\n", dir)

        err := os.Chdir(dir)
        if err != nil {
            abort("%s", err)
        }

        // parse all the templates we can find.
        filepath.Walk(".", func(filename string, info os.FileInfo, err error) error {

            if info != nil && info.IsDir() && filename != "." {
                return filepath.SkipDir
            }

            if info != nil && !info.IsDir() {
                printf("  template:%s\n", filename)

                g.Compile(filename)
            }

            return nil
        })

        printf("  executing:%s\n", name)

        err = g.Master.ExecuteTemplate(outputWriter, name, g.Scope.values)
        if err != nil {
            abort("%s", err)
        }
    }

    outputWriter.Close()

    //
    // Start server if requested
    //
    if *flgServer {
        go func() {
            dir := filepath.Dir(output)

            http.Handle("/", http.FileServer(http.Dir(dir)))

            printf("starting server for directory '%s' on http://127.0.0.1:5000/\n", dir)

            log.Fatal(http.ListenAndServe(":5000", nil))
        }()

        // start a browser. there is a race condition but the chances
        // of chrome or safari starting up before the server is tiny.
        url := "http://127.0.0.1:5000/" + filepath.Base(output)
        printf("opening %s\n", url)
        exec.Command("open", url).Run()

        // sleep forever
        select{}
    }
}
