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
#let author = "lemonilemon"
#let studentId = "123456789"

#set page(
  paper: "a4",
  margin: (x: 2cm, y: 3cm),
  numbering: "1",
  header: [
    #box(width: 1fr, context [
      #set align(left)
      #course
    ])
    #box(width: 1fr, context [
      #set align(center)
    ])
    #box(width: 1fr, context [
      #set align(right)
      #homework
    ])
    #box(fill: black, width: 100%, height: 1pt) // Top bar
  ],
  footer: context [
    #box(fill: black, width: 100%, height: 1pt)\
    #box(width: 1fr, context [
      #set align(left)
      Name: #author
    ])
    #box(width: 1fr, context [
      #set align(center)
      #counter(page).display("1")
    ])
    #box(width: 1fr, context [
      #set align(right)
      Student ID: #studentId
    ])

  ],
)

#set text(
  font: (
    (name: "CMU serif", covers: "latin-in-cjk"),
    "思源宋體",
  ),
  size: 12pt,
  cjk-latin-spacing: auto,
  lang: "en",
)

#set par(justify: true)
