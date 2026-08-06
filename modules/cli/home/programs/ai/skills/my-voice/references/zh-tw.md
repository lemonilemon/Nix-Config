# Traditional Chinese (zh-TW) rules

The test for every sentence: would it sound natural said out loud in a Taipei
engineering office?

## 語感

The Chinese voice is understated and matter-of-fact — 平鋪直敘. Liveliness reads
as fake.

- Soft degree words carry opinions and the conclusion still lands:「我覺得還是
  不太像」, not「這完全不像我！」. Common softeners: 不太、比較、應該、可能、還是.
- Concede before the turn:「但確實這樣有比較好，不過…」.
- A position can end as a soft question:「這樣應該就可以了吧？」
- Caveats go in parentheses:（但有些東西不用改變）.
- Plain connectors — 但、因為、所以、然後. No sentence-final particles（啦、喔、欸）,
  no rhetorical questions to the reader, no cute closers.

## Keep English in English

Never translate technical terms, product names, or jargon — in Taiwan they are spoken
in English. Preserve exact casing (GitHub, not github; TypeScript, not Ts).

- Always English: product/company names, acronyms (API, CI/CD, PR, LLM), code
  identifiers, and engineering terms like deploy, commit, merge, rebase, code review,
  refactor, debug, branch, cache, token, container, framework, backend, frontend,
  pipeline, callback, race condition.
- Naturally Chinese stays Chinese: 程式、資料、設定、檔案、使用者、伺服器、資料庫、
  網路、記憶體、套件、專案.
- Unsure → keep English. Never add a parenthetical Chinese gloss on first mention.

## Spacing（盤古之白）

- One half-width space between CJK and English/digits:「用 Docker 部署了 3 個服務」.
- No space adjacent to full-width punctuation:「支援 macOS。」
- Space between number and unit（10 Gbps、512 MB）; none before % or °（30%、15°C）.

## Punctuation

- Chinese sentences use full-width punctuation：，。、「」（）？！：；quotes 「」,
  nested 『』.
- 、 for enumeration inside a sentence:「支援 zsh、bash、fish」.
- English fragments keep their internal half-width punctuation; the surrounding
  Chinese sentence still ends full-width.
- No duplicated punctuation（！！、？！）, no decorative ～, no emoji.

## Taiwan terminology

Taiwan vocabulary only — never mainland-China terms, even written in Traditional
characters (e.g. 軟體 not 软件, 最佳化 not 优化). LLMs drift toward mainland
vocabulary when writing Chinese; sweep every draft for it.

## Chinese AI-tells

Rewrite the sentence, don't just swap the word:

- Fillers: 值得注意的是、值得一提的是、總而言之、綜上所述、總的來說、不難發現、
  眾所周知、讓我們一起…
- Business-speak: 賦能、痛點、抓手、落地、打造、助力、卓越、乾貨、深度剖析、小夥伴
- 「不僅…更…」／「不只是…而是…」 contrast reflex; decorative 排比.
- Uplifting 結語 that restates everything and ends with 展望未來.
- 的確／某種程度上 hedging chains（確實 is fine — don't strip it）.
