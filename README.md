Gossip
======
Gossip integrates [Go](http://golang.org) templates with a markdown to HTML processor. It is a command line utility that extends the Go template language with additional processing functions explained below.

Its sweetspot is generating single page HTML documentation for projects.  Using the [template language](http://golang.org/pkg/text/template/) of Go you can integrate multiple CSS, JS and markdown documents into a single file.


# Install
```
go get github.com/kuwerty/gossip
```

# Usage
The example can be compiled with

```
gossip -server -D Title=test -o example/public/index.html example/index.html example/roryg-ghostwriter.css example/test.md
```

All arguments listed on the command line are compiled as templates. The template names are set to the basename of the corresponding source file. The first file listed is then used as the entry point for template execution.

A -D option (can be used multiple times) defines a key=value pair for the initial template context. In this case we set 'Title' to 'test' which the template uses to set the HTML title of the page.

If the -server option is supplied then after compilation Gossip starts an HTTP server on port 5000 that serves up the contents of the public directory for viewing.

# Markdown Function
gossip extends the template language with a function named 'markdown':

```
{{ markdown "rubberstamp.md" }}
```

The markdown function first expands any templates in the markdown and then uses the [blackfriday](https://github.com/russross/blackfriday) markdown parser to transform the result into HTML.

The markdown function also supports arguments as described in the macro function below.

Some arguments are recognized by the markdown function. 'toc=true' will direct the markdown processor to generate a TOC automatically.

# Macro Function
gossip extends the template language with a function named 'macro'.

It works like the builtin template function but takes a variable list of "key=value" arguments, e.g.
```
{{ macro "snippet.html" "text=foo" "class=big" }}
```

Inside snippet.html the arguments are available to the template, e.g.
```
<div class={{.class}}>
{{.text}}
</div>
```

macro can use any valid template name so an alternative is to use it in conjuction with the `define` keyword:
```
{{define "snippet"}}
<div class={{.class}}>
{{.text}}
</div>

{{ macro "snippet" "text=important" "class=big" }}
{{ macro "snippet" "text=not so much" "class=small" }}
```



# Include Function
gossip extends the template language with a function named 'include'.

The argument is included directly without any template processing.

```
<div class={{.class}}>
{{include "some_content.html"}}
</div>
```



# Closure Function
gossip extends the template language with a function named 'closure'.

This command invokes the [closure-compiler](https://developers.google.com/closure/compiler/).

The arguments are joined together with spaces and used as the arguments
to invoke closure-compiler. The compiler must be on the path.

The arguments should not specify an output file, we will collect the output of closure and
use it.

Assuming html delimiters are enabled (--html) and the following switches between invoking
closure to compile the javascript or invoking closure to generate a minified, concatenated
version of the javascript.

```
<!-- if CLOSURE -->
<!-- clojure "--js jquery.js --js app.js" -->
<!-- else -->
<script ... src="jquery.js"></script>
<script ... src="app.js"></script>
<!-- end -->
```


