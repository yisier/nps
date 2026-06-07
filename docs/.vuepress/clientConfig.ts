import { defineClientConfig } from 'vuepress/client'
import { watch } from 'vue'
import { usePageData, useRoute } from 'vuepress/client'

export default defineClientConfig({
  setup() {
    const page = usePageData()
    const route = useRoute()
    // eslint-disable-next-line no-console
    watch(() => route.path, () => {
      // eslint-disable-next-line no-console
      console.log('[debug] route:', route.path, 'headers:', JSON.stringify(page.value.headers))
    }, { immediate: true })
  },
})
