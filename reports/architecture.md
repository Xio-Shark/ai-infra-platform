# 架构笔记

MVP 保持单进程可运行，但内部仍按 API / 调度 / 执行 / 观测分层，方便后续拆成独立服务。核心收益：

- 演示 training / inference / benchmark 三类 AI 作业的统一建模
- 用优先级队列说明调度策略
- 用执行记录与 trace timeline 说明可观测性闭环
