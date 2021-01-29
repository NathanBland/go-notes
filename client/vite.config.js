import vue from '@vitejs/plugin-vue'
import Pages from 'vite-plugin-pages';

export default {
  plugins: [vue({
    include: [/\.vue$/],
  }), Pages({
    extensions: ['vue']
  })]
}
