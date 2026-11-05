// Package icons 图标提取与声明（1016 起 .app → icns → PNG；1040 增补任意源
// 提取：本地图片文件 / http(s) URL 含 favicon.ico，见 source.go），供配置页
// 「提取」入口使用。
//
// 提取结果是缩放到 ≤64px 的 PNG 字节，调用方 base64 后存入 icon 声明。
// 纯函数、无环境副作用：只依赖入参路径/URL 与字节运算（URL 提取按入参显式外呼）。
// 非 macOS 平台的 .app 包结构相同，本包同样可用（「darwin-only」指使用场景，
// 无需 build tag）。
package icons
