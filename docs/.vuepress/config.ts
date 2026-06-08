import type { SidebarConfig } from 'vuepress'
import { defineUserConfig } from 'vuepress'
import { viteBundler } from '@vuepress/bundler-vite'
import { defaultTheme } from '@vuepress/theme-default'
import { prismjsPlugin } from '@vuepress/plugin-prismjs'
import { searchPlugin } from '@vuepress/plugin-search'

const serverSidebar: SidebarConfig = [
  { text: '服务端', children: [
    { text: '使用说明', link: '/server/nps_use.html' },
    { text: '配置文件', link: '/server/server_config.html' },
    { text: '增强功能', link: '/server/nps_extend.html' },
  ]},
]

const clientSidebar: SidebarConfig = [
  { text: '客户端', children: [
    { text: '使用说明', link: '/client/use.html' },
    { text: '增强功能', link: '/client/npc_extend.html' },
    { text: 'SDK', link: '/client/npc_sdk.html' },
  ]},
]

const extendSidebar: SidebarConfig = [
  { text: '扩展', children: [
    { text: '更多功能', link: '/extend/feature.html' },
    { text: '配置示例', link: '/extend/example.html' },
    { text: 'API接入方式', link: '/extend/api.html' },
    { text: 'API 清单', link: '/extend/webapi.html' },
  ]},
]

export default defineUserConfig({
  base: '/nps/',
  bundler: viteBundler(),
  lang: 'zh-CN',
  title: 'NPS',
  description: 'NPS 内网穿透文档 — 服务端、客户端、Web API、增强功能、更新日志',
  head: [
    ['link', { rel: 'icon', href: '/logo.svg' }],
    ['meta', { name: 'theme-color', content: '#4f46e5' }],
  ],

  plugins: [
    prismjsPlugin({
      lineNumbers: true,
    }),
    searchPlugin({}),
  ],

  theme: defaultTheme({
    logo: '/logo.svg',
    repo: 'yisier/nps',
    repoLabel: 'GitHub',
    docsRepo: 'https://github.com/yisier/nps',
    docsBranch: 'master',
    docsDir: 'docs',
    editLink: true,
    editLinkText: '在 GitHub 上编辑此页',
    lastUpdated: true,
    lastUpdatedText: '最后更新',
    contributors: false,

    navbar: [
      { text: '首页', link: '/' },
      { text: '快速上手', link: '/install/' },
      {
        text: '服务端',
        children: [
          { text: '使用说明', link: '/server/nps_use.html' },
          { text: '配置文件', link: '/server/server_config.html' },
          { text: '增强功能', link: '/server/nps_extend.html' },
        ],
      },
      {
        text: '客户端',
        children: [
          { text: '使用说明', link: '/client/use.html' },
          { text: '增强功能', link: '/client/npc_extend.html' },
          { text: 'SDK', link: '/client/npc_sdk.html' },
        ],
      },
      {
        text: '扩展',
        children: [
          { text: '更多功能', link: '/extend/feature.html' },
          { text: '配置示例', link: '/extend/example.html' },
          { text: 'API接入方式', link: '/extend/api.html' },
          { text: 'API 清单', link: '/extend/webapi.html' },
        ],
      },
      { text: '更新日志', link: '/changelog/' },
    ],

    sidebar: {
      '/install/': [
        { text: '快速上手', link: '/install/' },
      ],
      '/server/introduction.html': serverSidebar,
      '/server/nps_use.html': serverSidebar,
      '/server/server_config.html': serverSidebar,
      '/server/nps_extend.html': serverSidebar,
      '/client/use.html': clientSidebar,
      '/client/npc_extend.html': clientSidebar,
      '/client/npc_sdk.html': clientSidebar,
      '/extend/feature.html': extendSidebar,
      '/extend/example.html': extendSidebar,
      '/extend/api.html': extendSidebar,
      '/extend/webapi.html': extendSidebar,
      '/': [
        { text: '首页', link: '/' },
        { text: '快速上手', link: '/install/' },
        {
          text: '服务端',
          children: [
            { text: '使用说明', link: '/server/nps_use.html' },
            { text: '配置文件', link: '/server/server_config.html' },
            { text: '增强功能', link: '/server/nps_extend.html' },
          ],
        },
        {
          text: '客户端',
          children: [
            { text: '使用说明', link: '/client/use.html' },
            { text: '增强功能', link: '/client/npc_extend.html' },
            { text: 'SDK', link: '/client/npc_sdk.html' },
          ],
        },
        {
          text: '扩展',
          children: [
            { text: '功能', link: '/extend/feature.html' },
            { text: '配置示例', link: '/extend/example.html' },
            { text: 'API接入方式', link: '/extend/api.html' },
            { text: 'Web API 详细', link: '/extend/webapi.html' },
          ],
        },
        {
          text: '其他',
          children: [
            { text: 'FAQ', link: '/other/faq.html' },
            { text: '贡献', link: '/other/contribute.html' },
            { text: '讨论', link: '/other/discuss.html' },
            { text: '捐赠', link: '/other/donate.html' },
            { text: '致谢', link: '/other/thanks.html' },
          ],
        },
        { text: '更新日志', link: '/changelog/' },
      ],
    },
    sidebarDepth: 2,
  }),
})
