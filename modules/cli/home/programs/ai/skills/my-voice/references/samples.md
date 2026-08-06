# Voice samples

Verbatim excerpts of lemonilemon's own writing, by format — the strongest voice
signal available. Match the rhythm and word choice of the closest excerpts; they
outrank every rule in SKILL.md.

- Verbatim only; paraphrasing destroys the cadence this file exists to capture.
- 3-10 sentences per excerpt; representativeness beats volume.
- Best candidates: drafts the user edited and then accepted. Propose additions as
  repo edits; the user rebuilds.

## Reports (English)

(none yet)

## Articles / posts (English)

(none yet)

## Reports（中文）

Introducing the my-voice skill — formal register (agent draft accepted 2026-08,
calibration round 2):

> my-voice 是一個讓 coding agent 模仿我個人寫作風格的 skill，目標是讓產出的文字讀
> 起來像我本人所寫，而不是一眼就能看出是 AI 生成。
>
> 規則分成三個部分。第一是中文書寫慣例：專有名詞與技術術語保留英文，不翻譯（例如
> token 不翻成「權杖」），中英文之間加半形空格，標點使用全形，用語以台灣慣用為準。
> 第二是一份 AI 慣用語清單，草稿完成後逐句檢查，將「值得注意的是」這類句式改寫。
> 第三是 samples.md，收錄我實際寫過或修改過的段落；由於規則難以完整描述語感，長期
> 仍需依賴樣本累積來提高相似度。
>
> 整個 skill 以 Nix 管理，放在 ~/nixos-config 中，修改規則後重新 build 即可生效。
> 目前樣本數量仍少，實際效果需要持續使用一段時間後才能評估。

## Articles / posts（中文）

Introducing the my-voice skill — casual register (agent draft accepted 2026-08,
calibration round 2):

> 我寫了一個叫 my-voice 的 skill，用來讓 coding agent 寫出來的東西比較像我自己寫
> 的，而不是一看就是 AI。
>
> 主要的問題在中文。請 agent 用中文寫報告的時候，它會把術語都翻成中文，比如 token
> 翻成「權杖」，但這些詞我平常都是直接講英文的，翻過來反而不自然。所以規則裡直接寫
> 清楚：專有名詞保留英文、中英文之間加空格、標點用全形、用台灣的用語。另外有一份清
> 單列了常見的 AI 腔，像「值得注意的是」這種，寫完會掃一遍改掉。
>
> 它還有一個 samples.md，用來存我寫過或改過的段落。規則本身抓不太到語感，應該還是
> 要靠累積樣本，讓它慢慢變得比較像。檔案都放在 ~/nixos-config 裡用 Nix 管理，要改
> 就改檔案再 rebuild，跟其他設定一樣。目前樣本還很少，所以到底像不像，可能要再多用
> 一陣子才知道。
