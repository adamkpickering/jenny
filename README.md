# jenny

> How does this work again?

\- *Me, every time I try to update my [Hugo](https://gohugo.io/)-based
personal website*

`jenny` is a static site generator for people who want to build their own
website, but who also want to write its pages in markdown. It strives to be easy
to understand so that infrequent updates aren't painful. It uses the
[Go templating language](https://pkg.go.dev/text/template), and features
a `serve` subcommand with hot reloading for convenient development. Statically
linked binaries are provided for easy installation and uninstallation.


## How does it work?

The core of `jenny` is `jenny build`. When you run `jenny build`, `jenny`
recursively iterates over the files in `content/`. For each file, its
action depends on the extension of the file:

1. If the file has the `.md` extension, its [front matter](#front-matter)
   is stripped and parsed, and then the rest of the file is converted
   from markdown to HTML. The template referred to by the `TemplateName`
   field of the front matter is then filled using the compiled markdown
   and other context as input data. The result is written to the same
   relative path in `output/`, with the file extension changed to `.html`.

2. If the file does not have the `.md` extension, it is simply copied
   to the same relative path in `output/`. This is useful for static files.


### Project Structure

`content/` contains `.md` files that get compiled to HTML to make the
pages of your website. May also contain non-markdown files that you want
to include in your website (for example static files). May contain
nested directories.

`templates/` contains template files that specify how the HTML produced from
compiling markdown files is used in your website. This directory must be flat:
files in nested directories are not considered. Only files with the `.gotmpl`
extension are considered.

`output/` contains your built website. Its structure mirrors the structure
of `content/`, but with `.md` files renamed to `.html` files.

> [!NOTE]
> The paths listed for these directories are the defaults, but you may
> change them using a [`configuration.yaml`](#configurationyaml) file at
> the root of your project.


### Example Site

See [`example/`](example) for an example of a `jenny`-based website.


## Reference

### Front Matter

Front matter is a concept that is [copied from Hugo](https://gohugo.io/content-management/front-matter/).
It is a YAML-formatted preamble to the main markdown content that provides
`jenny` with metadata about a page when building your site. These are
the front matter fields that `jenny` supports:

| Field | Required | Description |
| --- | --- | --- |
| `LastModified` | no | The date the page was last modified |
| `Published` | no | The date the page was originially published |
| `Title` | no | The title of the page |
| `TemplateName` | yes | The name of the template used to build this page |


### `configuration.yaml`

`configuration.yaml` contains project-wide configuration. These are the fields
that it supports:

| Field | Description |
| --- | --- |
| `Content` | The path to the content directory |
| `Output` | The path to the output directory |
| `Templates` | The path to the templates directory |


### Data Available to Templates

You can see what data is available to templates for a specific content file
using the `jenny template-data` command. Using the [example site](example), the
output of `jenny template-data content/post1.md` is shown below (with comments
added):

```yaml
# Data pertaining to the content file you passed as an argument.
Page:
    # The HTML created from the markdown content.
    Content: redacted for legibility
    # Metadata from the front matter in the content file. For specifics please
    # see the Front Matter reference.
    Metadata:
        LastModified: 2024-03-24T00:00:00Z
        Published: 2024-02-15T00:00:00Z
        TemplateName: page.gotmpl
        Title: Post One
    # The path to the built page. Useful for linking.
    Path: /post1.html
    # The unmodified markdown content.
    RawContent: redacted for legibility
    # The path to the content file.
    SourcePath: content/post1.md

# Data for all the pages in the site. Elements are the same as the Page key.
Pages:
    - Content: redacted for legibility
      Metadata:
        TemplateName: index.gotmpl
      Path: /index.html
      RawContent: redacted for legibility
      SourcePath: content/index.md
    - Content: redacted for legibility
      Metadata:
        LastModified: 2024-03-24T00:00:00Z
        Published: 2024-02-15T00:00:00Z
        TemplateName: page.gotmpl
        Title: Post One
      Path: /post1.html
      RawContent: redacted for legibility
      SourcePath: content/post1.md
    - Content: redacted for legibility
      Metadata:
        LastModified: 2024-04-25T00:00:00Z
        Published: 2024-03-16T00:00:00Z
        TemplateName: page.gotmpl
        Title: Post Two
      Path: /post2.html
      RawContent: redacted for legibility
      SourcePath: content/post2.md

# Project-wide data
Context:
    # The datetime for when `jenny` was run.
    Now: 2024-11-19T12:36:34.635218497-07:00
    # The contents of configuration.yaml. For specifics please see the
    # configuration.yaml reference.
    Config:
        Content: content
        Output: output
        Templates: templates
```

## How does hot reloading work?

In order to make development of your site as easy as possible, `jenny`
has a `serve` subcommand that rebuilds your site every time a file in
`content/` or `templates/` is changed. Here this is referred to as
"hot reloading".

`jenny` uses websockets for this. On startup and each time a change is
detected, `jenny` builds the site like it would for the `build` subcommand,
and then injects a script into each HTML file in `output/`. The script
opens a websocket against the `/websocket` server endpoint and listens
for messages; when a message is received, it reloads the page.
The server listens on `/websocket`, and sends a message on this websocket
each time a change is detected (but only after the rebuild is completed).

This scheme works pretty well. However, it is possible that an error
causes the server to exit, which means hot reloading stops. It is easy
for the user to miss this. A few ways of notifying the user or recovering
gracefully have been explored, but none are satisfactory - the simplest
solution is to rely on the user to notice the failure and restart `jenny`.


## Credits

Thanks to SUSE for holding [Hack Week] 24, which helped to get `jenny`
to the point where it is usable!
