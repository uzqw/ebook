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

  it('segments paragraphs by short final lines when extraction has no blank lines', () => {
    expect(
      reflowText(
        '例如支付系统是 0 级系统，而优惠券是 1 级系统的话，在极端情况下可以把优惠券给降\n' +
          '级，防止支付系统被优惠券这个 1 级系统给拖垮。\n' +
          '5. 不要有单点\n' +
          '系统中的单点可以说是系统架构上的一个大忌，因为单点意味着没有备份，风险不可控，\n' +
          '我们设计分布式系统最重要的原则就是“消除单点”。',
      ),
    ).toEqual([
      {
        kind: 'paragraph',
        text: '例如支付系统是 0 级系统，而优惠券是 1 级系统的话，在极端情况下可以把优惠券给降级，防止支付系统被优惠券这个 1 级系统给拖垮。',
      },
      { kind: 'paragraph', text: '5. 不要有单点' },
      {
        kind: 'paragraph',
        text: '系统中的单点可以说是系统架构上的一个大忌，因为单点意味着没有备份，风险不可控，我们设计分布式系统最重要的原则就是“消除单点”。',
      },
    ])
  })

  it('segments paragraphs by short final lines when EPUB blanks are synthetic', () => {
    expect(
      reflowText(
        '因為，斷捨離的奧義並非在此。\n\n' +
          '櫥櫃裏、餐架上或者冰箱中囤積的無用之物，家裏隨處堆積的廢品破爛，\n\n' +
          '還包括精神層面上那些不適宜的過剩觀念，或是讓自己陷入自我否定、\n\n' +
          '能解放自己、解放人生。',
      ),
    ).toEqual([
      {
        kind: 'paragraph',
        text: '因為，斷捨離的奧義並非在此。',
      },
      {
        kind: 'paragraph',
        text: '櫥櫃裏、餐架上或者冰箱中囤積的無用之物，家裏隨處堆積的廢品破爛，還包括精神層面上那些不適宜的過剩觀念，或是讓自己陷入自我否定、能解放自己、解放人生。',
      },
    ])
  })

  it('does not close a paragraph on a comma-ending short line', () => {
    expect(
      reflowText(
        '系統中的單點可以說是系統架構上的一個大忌，因為單點意味著沒有備份，\n' +
          '風險不可控，\n' +
          '我們設計分佈式系統最重要的原則就是「消除單點」。',
      ),
    ).toEqual([
      {
        kind: 'paragraph',
        text: '系統中的單點可以說是系統架構上的一個大忌，因為單點意味著沒有備份，風險不可控，我們設計分佈式系統最重要的原則就是「消除單點」。',
      },
    ])
  })

  it('returns no blocks when a scanned page has no extracted text', () => {
    expect(reflowText(' \n\f ')).toEqual([])
  })
})
