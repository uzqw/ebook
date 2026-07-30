export interface ReflowBlock {
  text: string
  kind: 'heading' | 'paragraph'
}

export type LayoutMode = 'original' | 'reflow'
export type LayoutPreference = LayoutMode | 'auto'

export function parseLayoutPreference(value: string | null): LayoutPreference {
  return value === 'original' || value === 'reflow' ? value : 'auto'
}

export function resolveLayoutMode(
  preference: LayoutPreference,
  narrowViewport: boolean,
): LayoutMode {
  return preference === 'auto' ? (narrowViewport ? 'reflow' : 'original') : preference
}

const headingPattern =
  /^(?:第[零〇一二三四五六七八九十百千万两\d]+[章节卷篇部回]|(?:chapter|part|section|book)\s+[\wivxlcdm]+|序(?:章|言)?|前言|引言|楔子|尾声|后记|目录)(?:\s|$|[：:])/i
const cjkPattern = /[\u3040-\u30ff\u3400-\u9fff\uac00-\ud7af]/
const noSpaceBeforePattern = /^[,.;:!?，。！？；：、）】」』》”’]/
const noSpaceAfterPattern = /[(（【「『《“‘—]$/

function isHeading(text: string) {
  return text.length <= 80 && headingPattern.test(text)
}

function joinLines(left: string, right: string) {
  if (!left) return right
  if (!right) return left

  if (/[A-Za-z]-$/.test(left) && /^[a-z]/.test(right)) {
    return `${left.slice(0, -1)}${right}`
  }

  const last = left[left.length - 1] || ''
  const first = right[0] || ''
  if (
    cjkPattern.test(last) ||
    cjkPattern.test(first) ||
    noSpaceAfterPattern.test(left) ||
    noSpaceBeforePattern.test(right)
  ) {
    return `${left}${right}`
  }
  return `${left} ${right}`
}

function hasSyntheticLineSpacing(lines: string[]) {
  const contentIndexes = lines.flatMap((line, index) => (line.trim() ? [index] : []))
  if (contentIndexes.length < 3) return false

  let singleBlankGaps = 0
  for (let index = 1; index < contentIndexes.length; index++) {
    if (contentIndexes[index] - contentIndexes[index - 1] === 2) singleBlankGaps++
  }

  // MuPDF's EPUB extraction commonly emits `visual line\n\nvisual line` for
  // almost every rendered line. Those empty lines are layout artifacts, not
  // paragraph boundaries, and preserving them cuts sentences into fragments.
  return singleBlankGaps / (contentIndexes.length - 1) >= 0.75
}

/**
 * Turns fixed-page text extraction into fluid paragraphs. Blank lines and
 * indented lines remain paragraph boundaries; visual line wraps are joined so
 * the browser can wrap them again for the current viewport.
 */
export function reflowText(rawText?: string): ReflowBlock[] {
  if (!rawText?.trim()) return []

  const lines = rawText.replace(/\r\n?/g, '\n').replace(/\f/g, '\n\n').split('\n')
  const syntheticLineSpacing = hasSyntheticLineSpacing(lines)
  const blocks: ReflowBlock[] = []
  let paragraph = ''

  const flushParagraph = () => {
    const text = paragraph.trim()
    if (text) blocks.push({ text, kind: isHeading(text) ? 'heading' : 'paragraph' })
    paragraph = ''
  }

  for (const sourceLine of lines) {
    const text = sourceLine.trim()
    if (!text) {
      if (!syntheticLineSpacing) flushParagraph()
      continue
    }

    if (isHeading(text)) {
      flushParagraph()
      blocks.push({ text, kind: 'heading' })
      continue
    }

    const startsParagraph = /^[\t ]{2,}|^\u3000+/.test(sourceLine)
    if (startsParagraph) flushParagraph()
    paragraph = joinLines(paragraph, text)
  }
  flushParagraph()

  return blocks
}
