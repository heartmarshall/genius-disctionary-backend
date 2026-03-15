import type { PartOfSpeech } from '@/types/dictionary'

/**
 * Maps part-of-speech to Herbarium palette color tokens.
 * Returns Tailwind class fragments for badge bg, text, and border.
 */
type PosColorScheme = {
  bg: string
  text: string
  border: string
  dot: string
  /** CSS variable name for the raw color value (e.g. '--cornflower') */
  cssVar: string
}

const POS_COLORS: Record<string, PosColorScheme> = {
  NOUN: { bg: 'bg-cornflower-light', text: 'text-cornflower-fg', border: 'border-cornflower/20', dot: 'bg-cornflower', cssVar: '--cornflower' },
  VERB: { bg: 'bg-poppy-light', text: 'text-poppy-fg', border: 'border-poppy/20', dot: 'bg-poppy', cssVar: '--poppy' },
  ADJECTIVE: { bg: 'bg-goldenrod-light', text: 'text-goldenrod-fg', border: 'border-goldenrod/20', dot: 'bg-goldenrod', cssVar: '--goldenrod' },
  ADVERB: { bg: 'bg-thyme-light', text: 'text-thyme-fg', border: 'border-thyme/20', dot: 'bg-thyme', cssVar: '--thyme' },
  PRONOUN: { bg: 'bg-cornflower-light', text: 'text-cornflower-fg', border: 'border-cornflower/20', dot: 'bg-cornflower', cssVar: '--cornflower' },
  PREPOSITION: { bg: 'bg-goldenrod-light', text: 'text-goldenrod-fg', border: 'border-goldenrod/20', dot: 'bg-goldenrod', cssVar: '--goldenrod' },
  CONJUNCTION: { bg: 'bg-thyme-light', text: 'text-thyme-fg', border: 'border-thyme/20', dot: 'bg-thyme', cssVar: '--thyme' },
  INTERJECTION: { bg: 'bg-poppy-light', text: 'text-poppy-fg', border: 'border-poppy/20', dot: 'bg-poppy', cssVar: '--poppy' },
  PHRASE: { bg: 'bg-source-book-light', text: 'text-source-book', border: 'border-source-book/20', dot: 'bg-source-book', cssVar: '--source-book' },
  IDIOM: { bg: 'bg-source-music-light', text: 'text-source-music', border: 'border-source-music/20', dot: 'bg-source-music', cssVar: '--source-music' },
  OTHER: { bg: 'bg-surface-secondary', text: 'text-text-secondary', border: 'border-border-subtle', dot: 'bg-text-secondary', cssVar: '--border-default' },
}

const DEFAULT_COLORS: PosColorScheme = POS_COLORS.OTHER

export function getPosColors(pos: PartOfSpeech | string | undefined | null): PosColorScheme {
  if (!pos) return DEFAULT_COLORS
  return POS_COLORS[pos] ?? DEFAULT_COLORS
}

/**
 * Returns the accent color stripe class for the left border of a card,
 * based on the primary POS of the entry.
 */
export function getPosAccentBorder(pos: PartOfSpeech | string | undefined | null): string {
  if (!pos) return 'border-l-border-default'
  const map: Record<string, string> = {
    NOUN: 'border-l-cornflower',
    VERB: 'border-l-poppy',
    ADJECTIVE: 'border-l-goldenrod',
    ADVERB: 'border-l-thyme',
    PRONOUN: 'border-l-cornflower',
    PREPOSITION: 'border-l-goldenrod',
    CONJUNCTION: 'border-l-thyme',
    INTERJECTION: 'border-l-poppy',
    PHRASE: 'border-l-source-book',
    IDIOM: 'border-l-source-music',
    OTHER: 'border-l-border-default',
  }
  return map[pos] ?? 'border-l-border-default'
}
