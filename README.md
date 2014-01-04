Gossip
======
Gossip integrates [Go](http://golang.org) templates with a markdown to HTML processor.

Its sweetspot is generating single page HTML documentation for Go projects (where the output of godoc might not work very well).  Using the [template language](http://golang.org/pkg/text/template/) of Go you can integrate multiple CSS, JS and markdown documents into a single file. Maybe for publishing on github-pages.

Gossip is a command line utility that extends the Go template language with additional markdown processing functions documented below.

# Install
```
go get github.com/kuwerty/gossip
```

# Usage
Invoke with
```
gossip -server source/index.html >public/index.html
```

Gossip reads each file on the command line in turn.  All files found in the 'source' directory are parsed as template files and compiled. We then execute the template named 'index.html'. The output is written to stdout.

After compilation it starts a HTTP server on port 5000 that serves up the contents of the public directory for viewing.


# Markdown Function
gossip extends the template language with a function named 'markdown':

```
{{ markdown "rubberstamp.md" }}
```

The markdown function first expands any templates in the markdown and then uses the [blackfriday](https://github.com/russross/blackfriday) markdown parser to transform the result into HTML.

The markdown function also supports arguments as described in the macro function below.

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
