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
const numberedSectionPattern = /^(?:\d{1,3}(?:\.\d{1,3})+|\d{1,3}(?:\.\d{1,3})*[.、])\s*(.+)$/
const capitalizedWordPattern = /^[^A-Za-z]*[A-Z]/
const cjkPattern = /[\u3040-\u30ff\u3400-\u9fff\uac00-\ud7af]/
const fullWidthPattern =
  /[\u1100-\u11ff\u2e80-\u303f\u3040-\u30ff\u3400-\u9fff\uf900-\ufaff\uff00-\uffef]/
const paragraphEndPattern = /[.!?。！？…][）)】」』》"'’”]?$/
const sentenceContinuationPattern = /[-,，、;；:：]$/
const noSpaceBeforePattern = /^[,.;:!?，。！？；：、）】」』》”’]/
const noSpaceAfterPattern = /[(（【「『《“‘—]$/

function isHeading(text: string, { allowTitleCase = false } = {}) {
  return (
    (text.length <= 80 && headingPattern.test(text)) ||
    isNumberedSectionHeading(text) ||
    isAllCapsHeading(text) ||
    (allowTitleCase && isTitleCaseHeading(text))
  )
}

// Numbered sections such as "5. 不要有单点" or "3.1.2 不同场景下的不同架构案例".
// The remainder must open like a title (CJK or uppercase Latin) and the line
// must not end with sentence punctuation, so wrapped sentences opening with a
// quantity ("3.5 million people ...") stay body text.
function isNumberedSectionHeading(text: string) {
  if (
    text.length > 40 ||
    paragraphEndPattern.test(text) ||
    sentenceContinuationPattern.test(text)
  ) {
    return false
  }
  const match = numberedSectionPattern.exec(text)
  return !!match && /^[A-Z\u2e80-\u9fff\uff00-\uffef]/.test(match[1])
}

// ALL-CAPS English titles like "MY RECOVERY". CJK lines and anything with
// lowercase letters are excluded; several letters or multiple words are
// required so interjections like "OK" never qualify.
function isAllCapsHeading(text: string) {
  if (text.length > 60 || cjkPattern.test(text) || /[a-z]/.test(text)) return false
  const letters = text.match(/[A-Z]/g)?.length ?? 0
  if (letters < 2) return false
  return text.split(/\s+/).length >= 2 || letters >= 4
}

// Plain English titles like "The Power of Now": short, no sentence-final
// punctuation, mostly capitalized words. Only applied to standalone blocks —
// a capitalized sentence fragment mid-paragraph ("Fluid Latin text ...")
// must stay body text.
function isTitleCaseHeading(text: string) {
  if (
    text.length < 3 ||
    text.length > 50 ||
    cjkPattern.test(text) ||
    paragraphEndPattern.test(text) ||
    sentenceContinuationPattern.test(text)
  ) {
    return false
  }
  const words = text.split(/\s+/).filter((word) => /[A-Za-z]/.test(word))
  if (words.length < 2) return false
  const capitalized = words.filter((word) => capitalizedWordPattern.test(word)).length
  return capitalized >= 2 && capitalized / words.length > 0.5
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

function displayWidth(text: string) {
  let width = 0
  for (const char of text) width += fullWidthPattern.test(char) ? 1 : 0.5
  return width
}

// Justified books fill every visual line to the right margin except each
// paragraph's final line. When extraction supplies no paragraph markers, a
// noticeably short line is the only segmentation signal left. Lines ending
// in a hyphen or comma continue on the next line, so they never close a
// paragraph.
function endsParagraphLine(text: string, maxLineWidth: number) {
  if (!maxLineWidth || sentenceContinuationPattern.test(text)) return false
  const ratio = displayWidth(text) / maxLineWidth
  return ratio <= 0.7 && (ratio <= 0.45 || paragraphEndPattern.test(text))
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
 * the browser can wrap them again for the current viewport. When extraction
 * provides no paragraph markers at all (PDF pages without blank lines, or
 * EPUB pages with a synthetic blank after every visual line), paragraph-final
 * short lines become the boundaries instead.
 */
export function reflowText(rawText?: string): ReflowBlock[] {
  if (!rawText?.trim()) return []

  const lines = rawText.trim().replace(/\r\n?/g, '\n').replace(/\f/g, '\n\n').split('\n')
  const syntheticLineSpacing = hasSyntheticLineSpacing(lines)
  const inferParagraphBreaks = syntheticLineSpacing || lines.every((line) => line.trim())
  const maxLineWidth = inferParagraphBreaks
    ? Math.max(0, ...lines.filter((line) => line.trim()).map((line) => displayWidth(line.trim())))
    : 0
  const blocks: ReflowBlock[] = []
  let paragraph = ''

  const flushParagraph = () => {
    const text = paragraph.trim()
    if (text)
      blocks.push({
        text,
        kind: isHeading(text, { allowTitleCase: true }) ? 'heading' : 'paragraph',
      })
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
    if (inferParagraphBreaks && endsParagraphLine(text, maxLineWidth)) flushParagraph()
  }
  flushParagraph()

  return blocks
}
