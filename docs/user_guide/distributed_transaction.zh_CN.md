## 1 背景

TDXA 是腾讯自研的一种分布式事务解决方案，相关信息见 km 文章：[腾讯分布式事务 Oteam 解决方案介绍](https://km.woa.com/articles/show/539638?kmref=search&from_page=1&no=6)

## 2 原理

TDXA 的整体技术原理见 [TDXA 总体技术方案 iwiki](https://iwiki.woa.com/pages/viewpage.action?pageId=905611335)。

其中涉及 trpc-go 的部分见 [Go SDK 方案](https://iwiki.woa.com/pages/viewpage.action?pageId=1426747917)，主要技术点为流式及 filter。

## 3 实现

见 TDXA 项目[代码仓库](https://git.woa.com/groups/TDXA/-/projects/list)

## 4 示例

见 [TDXA 用户使用文档](https://iwiki.woa.com/pages/viewpage.action?pageId=1441335912)

## 5 FAQ

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
