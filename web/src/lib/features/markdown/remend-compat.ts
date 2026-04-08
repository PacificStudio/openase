import remendDefault, {
  isWithinCodeBlock,
  isWithinLinkOrImageUrl,
  isWithinMathBlock,
  isWordChar,
} from '../../../../node_modules/.pnpm/node_modules/remend/dist/index.js'

type RemendHandler = {
  handle: (text: string) => string
  name: string
  priority?: number
}

type RemendOptions = {
  bold?: boolean
  boldItalic?: boolean
  comparisonOperators?: boolean
  handlers?: RemendHandler[]
  htmlTags?: boolean
  images?: boolean
  inlineCode?: boolean
  inlineKatex?: boolean
  italic?: boolean
  katex?: boolean
  linkMode?: 'protocol' | 'text-only'
  links?: boolean
  setextHeadings?: boolean
  singleTilde?: boolean
  strikethrough?: boolean
}

type Plugin = {
  handle: (text: string) => string
  name: string
  priority?: number
}

const DEFAULT_PLUGIN_NAMES = [
  'singleTilde',
  'comparisonOperators',
  'htmlTags',
  'setextHeadings',
  'links',
  'boldItalic',
  'bold',
  'doubleUnderscoreItalic',
  'singleAsteriskItalic',
  'singleUnderscoreItalic',
  'inlineCode',
  'strikethrough',
  'katex',
  'inlineKatex',
] as const

const defaultPluginHandle = (text: string) => remendDefault(text)

export class IncompleteMarkdownParser {
  readonly plugins: Plugin[]

  constructor(plugins: Plugin[] = IncompleteMarkdownParser.createDefaultPlugins()) {
    this.plugins = plugins
  }

  static createDefaultPlugins(): Plugin[] {
    return DEFAULT_PLUGIN_NAMES.map((name) => ({
      name,
      handle: defaultPluginHandle,
    }))
  }

  parse(text: string, options?: RemendOptions): string {
    return remendDefault(text, options)
  }
}

export type { Plugin, RemendHandler, RemendOptions }

export const parseIncompleteMarkdown = (text: string, options?: RemendOptions): string =>
  remendDefault(text, options)

export { isWithinCodeBlock, isWithinLinkOrImageUrl, isWithinMathBlock, isWordChar }

export default remendDefault
