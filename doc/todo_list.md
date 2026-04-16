# TODO List

1. Project Name 目前不支援用括號
主要原因是runtime 需要用project name去作為log的file name
需檢查其他符號是否也有機會出問題

2. Vite App 的 內建指令導致runtime start 按鈕卡死
具體原因不明, 但對laravel + react vite項目來說
project type選vite app, runtime 選 node.js會出問題
但runtime選custom, start command輸入npm run dev就可以了
可能原因: 主機沒安裝Node.js或Bun導致掃瞄卡死
參考: doc/runtime_start_btn_stuck_at_disabled_problem.md