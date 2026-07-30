import { describe, expect, it } from 'vitest'
import { parseLayoutPreference, reflowText, resolveLayoutMode } from './reflow'

describe('layout mode', () => {
  it('defaults narrow viewports to reflow and wider viewports to original pages', () => {
    expect(resolveLayoutMode(parseLayoutPreference(null), true)).toBe('reflow')
    expect(resolveLayoutMode(parseLayoutPreference(null), false)).toBe('original')
  })

  it('honors a valid explicit preference at either viewport width', () => {
    expect(resolveLayoutMode(parseLayoutPreference('original'), true)).toBe('original')
    expect(resolveLayoutMode(parseLayoutPreference('reflow'), false)).toBe('reflow')
  })

  it('treats an unknown saved preference as automatic', () => {
    expect(parseLayoutPreference('unknown')).toBe('auto')
  })
})

describe('reflowText', () => {
  it('joins extracted CJK visual lines without inserting spaces', () => {
    expect(reflowText('这是固定页面的第一行\n这是第二行')).toEqual([
      { kind: 'paragraph', text: '这是固定页面的第一行这是第二行' },
    ])
  })

  it('joins EPUB visual lines when extraction inserts a blank after every line', () => {
    expect(
      reflowText(
        '當一個人懷著同理心為他人著想時，他絕對不\n\n會孤單。開放的心是寂寞的解藥。\n\n只要敞開心胸就能發出溫暖。',
      ),
    ).toEqual([
      {
        kind: 'paragraph',
        text: '當一個人懷著同理心為他人著想時，他絕對不會孤單。開放的心是寂寞的解藥。只要敞開心胸就能發出溫暖。',
      },
    ])
  })

  it('also rejoins Latin EPUB lines separated by synthetic blank lines', () => {
    expect(
      reflowText('The sentence continues\n\non the following visual line\n\nwithout a cut.'),
    ).toEqual([
      {
        kind: 'paragraph',
        text: 'The sentence continues on the following visual line without a cut.',
      },
    ])
  })

  it('keeps real blank and indented lines as paragraph boundaries', () => {
    expect(reflowText('第一段\n继续第一段\n\n　　第二段')).toEqual([
      { kind: 'paragraph', text: '第一段继续第一段' },
      { kind: 'paragraph', text: '第二段' },
    ])
  })

  it('recognizes chapter headings and rejoins hyphenated English words', () => {
    expect(reflowText('Chapter 1\nAuto-\nmatically reflowed text.')).toEqual([
      { kind: 'heading', text: 'Chapter 1' },
      { kind: 'paragraph', text: 'Automatically reflowed text.' },
    ])
  })

  it('adds spaces between wrapped Latin lines without spaces before punctuation', () => {
    expect(reflowText('Fluid Latin text\nwraps again\n, with punctuation.')).toEqual([
      { kind: 'paragraph', text: 'Fluid Latin text wraps again, with punctuation.' },
    ])
  })

  it('returns no blocks when a scanned page has no extracted text', () => {
    expect(reflowText(' \n\f ')).toEqual([])
  })
})
