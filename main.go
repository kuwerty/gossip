package main

import (
    "text/template"
    "flag"
    "fmt"
    "log"
    "os"
    "os/exec"
    "bytes"
    "path"
    "strings"
    "strconv"
    "path/filepath"
    "github.com/russross/blackfriday"
    "net/http"
)

var useHtmlDelims = false


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

    if useHtmlDelims {
        if strings.HasSuffix(filename, ".html") || strings.HasSuffix(filename, ".htm") ||
           strings.HasSuffix(filename, ".xml") || strings.HasSuffix(filename, ".xhtml") {

            t.Delims("<!--", "-->")
        }
    }

    return template.Must(t.ParseFiles(filename))
}

func (g *Generator) Parse(name string, contents string) *template.Template {
    t := g.Master.New(name).Funcs(g.Funcs)

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

func (g *Generator) markdown(name string, argsv ...string) string {
    // find template with correct name
    t := g.Master.Lookup(name)
    if t == nil {
        abort("cannot find template '%s' (used as markdown function argument)", name)
    }

    // create new scope and execute the template
    args := g.ParseArgs(argsv)
    g.PushScope(args)
    defer g.PopScope()

    buf := bytes.NewBuffer(nil)
    err := t.Execute(buf, nil)
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
    basename  := path.Base(name)
    extension := path.Ext(basename)
    mdname    := basename[0:len(basename)-len(extension)]

    renderer := blackfriday.HtmlRenderer(htmlFlags, mdname, "")

    html := blackfriday.Markdown(buf.Bytes(), renderer, extensions)

    return string(html)
}


func (g *Generator) include(filename string) string {
    printf("include %s\n", filename)

    // ho ho ho (shell expansion)
    buf, err := exec.Command("bash", "-c", "cat "+filename).Output()
    if err != nil {
        abort("%s with %s", err, string(buf))
    }

    return string(buf)
}

func (g *Generator) closure(argsv ...string) string {
    cmdargs := strings.Join(argsv, " ")

    printf("closure-compiler %s\n", cmdargs)

    // use bash to invoke and split the args
    buf, err := exec.Command("bash", "-c", "closure-compiler "+cmdargs).Output()
    if err != nil {
        abort("%s with %s", err, string(buf))
    }

    return string(buf)
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

    flag.BoolVar(&useHtmlDelims, "html", false, "Switch HTML template delimiters to <!-- --> in HTML files.")

    flag.Var(&defines, "D", "define values for the initial context in form KEY=VALUE, e.g -D GOSSIP=1")

    flag.Parse()

    if output == "" {
        flag.PrintDefaults()

        abort("must specify output file")
    }

    outputWriter, err := os.Create(output)
    if err != nil {
        abort("cannot open output: %s", err)
    }

    pwd,_ := os.Getwd()

    g := new(Generator)

    g.Funcs = template.FuncMap {
        "macro"     : g.macro,
        "markdown"  : g.markdown,
        "include"   : g.include,
        "closure"   : g.closure,
        "GOSSIP"    : func() string { return "YES" },
    }

    g.Master = template.New("master").Funcs(g.Funcs)

    g.PushScope(g.ParseArgs(defines))

    for _,arg := range flag.Args() {
        printf("compiling:%s\n", arg)

        g.Compile(arg)
    }

    main := filepath.Base(flag.Arg(0))

    printf("executing:%s\n", main)

    err = g.Master.ExecuteTemplate(outputWriter, main, g.Scope.values)
    if err != nil {
        abort("%s", err)
    }

    defer g.PopScope()

    outputWriter.Close()

    //
    // Start server if requested
    //
    if *flgServer {
        go func() {
            dir := pwd + "/" + filepath.Dir(output)

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
