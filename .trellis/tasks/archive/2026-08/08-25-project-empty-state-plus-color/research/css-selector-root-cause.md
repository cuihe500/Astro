# CSS 选择器根因

- `web/src/components/Feedback.tsx:29` 的 `EmptyState` 将顶部 `Inbox` 图标作为容器直属子元素，并把操作按钮作为 `children` 渲染。
- `web/src/styles.css:192` 的 `.page-state-empty svg` 命中所有后代 SVG，因此按钮内的 `Plus` 被信息色覆盖。
- 项目和应用空状态复用该结构，修复点应放在共享 CSS 规则：只匹配直属 SVG，让按钮图标继续继承 `.button-primary` 的白色前景色。
- 这是单行样式修复，不需要修改组件、增加类名或新增测试；使用 `make frontend-check` 做回归校验。
