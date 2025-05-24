;; typst
#import "@preview/showybox:2.0.4": showybox
#import "@preview/cetz:0.3.4"
#import "@preview/codly:1.3.0": *
#import "@preview/codly-languages:0.1.1": *
#show: codly-init.with()
#import "@preview/theorion:0.3.3": *
#import cosmos.clouds: *
#show: show-theorion
// #import cosmos.simple: *
// #show: show-theorion

#let course = "CS101"
#let homework = "Homework"
#let author = {{_author_}}

#set page(
  paper: "a4",
  margin: (x: 2cm, y: 3cm),
  numbering: "1",
  header: [
    _ #course _
    #h(1fr)
    #homework
    #box(fill: black, width: 100%, height: 1pt) // Top bar
  ],
  footer: context [
    #box(fill: black, width: 100%, height: 1pt)
    #author
    #h(1fr)
    #counter(page).display("1")
  ],
)

#set text(
  font: (
    (name: "CMU serif", covers: "latin-in-cjk"),
    "Noto Serif CJK TC",
  ),
  size: 12pt,
  cjk-latin-spacing: auto,
  lang: "en",
)

#set par(justify: true)

{{_cursor_}}
