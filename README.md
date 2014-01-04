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
gossip -server -D Title=test -o example/public/index.html example/index.html
```

After the flags have been stripped, Gossip treats each argument on the command line in turn.  For each argument, any files found in the same directory are treated as template files and compiled. It then uses the named file as the root of template instantiation.

The -D option defines a key=value pair for the initial template context. In this case we set 'Title' to 'test' which the template uses to set the HTML title of the page.

If the -server option is supplied then after compilation Gossip starts an HTTP server on port 5000 that serves up the contents of the public directory for viewing.


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
