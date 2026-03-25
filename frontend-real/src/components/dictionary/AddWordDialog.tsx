import { useState, useEffect, useRef } from 'react'
import { toast } from 'sonner'
import { Search, ArrowLeft, Check } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { useCatalogSearch, useRefEntryPreview, useAddFromCatalog } from '@/hooks/useAddWord'
import { getPosColors } from '@/lib/pos-colors'
import type { RefEntry, RefSense } from '@/types/dictionary'

const POS_SHORT: Record<string, string> = {
  NOUN: 'noun', VERB: 'verb', ADJECTIVE: 'adj', ADVERB: 'adv',
  PRONOUN: 'pron', PREPOSITION: 'prep', CONJUNCTION: 'conj',
  INTERJECTION: 'interj', PHRASE: 'phrase', IDIOM: 'idiom', OTHER: 'other',
}

interface AddWordDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function AddWordDialog({ open, onOpenChange }: AddWordDialogProps) {
  const [step, setStep] = useState<'search' | 'preview'>('search')
  const catalogSearch = useCatalogSearch()
  const refPreview = useRefEntryPreview()
  const { addFromCatalog, loading: adding } = useAddFromCatalog()
  const [selectedSenseIds, setSelectedSenseIds] = useState<Set<string>>(new Set())
  const [notes, setNotes] = useState('')

  const reset = () => {
    setStep('search')
    catalogSearch.setQuery('')
    setSelectedSenseIds(new Set())
    setNotes('')
  }

  const handleSelectResult = (entry: RefEntry) => {
    refPreview.preview(entry.text)
    setStep('preview')
  }

  // When preview loads, select all senses by default
  useEffect(() => {
    if (refPreview.entry) {
      setSelectedSenseIds(new Set(refPreview.entry.senses.map(s => s.id)))
    }
  }, [refPreview.entry])

  const toggleSense = (senseId: string) => {
    setSelectedSenseIds(prev => {
      const next = new Set(prev)
      if (next.has(senseId)) next.delete(senseId)
      else next.add(senseId)
      return next
    })
  }

  const handleAdd = async () => {
    if (!refPreview.entry || selectedSenseIds.size === 0) return

    try {
      await addFromCatalog(
        refPreview.entry.id,
        Array.from(selectedSenseIds),
        notes || undefined,
      )
      toast.success(`"${refPreview.entry.text}" added to dictionary`)
      reset()
      onOpenChange(false)
    } catch {
      toast.error('Failed to add word')
    }
  }

  const handleBack = () => {
    setStep('search')
    setSelectedSenseIds(new Set())
    setNotes('')
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { onOpenChange(v); if (!v) reset() }}>
      <DialogContent className="max-w-lg max-h-[85vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {step === 'preview' && (
              <button
                type="button"
                onClick={handleBack}
                className="p-1 -ml-1 rounded hover:bg-surface-secondary transition-colors"
              >
                <ArrowLeft size={16} />
              </button>
            )}
            {step === 'search' ? 'Add word' : refPreview.entry?.text ?? 'Loading...'}
          </DialogTitle>
          <DialogDescription className="sr-only">
            Search the reference dictionary and add a word
          </DialogDescription>
        </DialogHeader>

        {step === 'search' && (
          <SearchStep
            query={catalogSearch.query}
            onQueryChange={catalogSearch.setQuery}
            results={catalogSearch.results}
            loading={catalogSearch.loading}
            onSelect={handleSelectResult}
          />
        )}

        {step === 'preview' && (
          <PreviewStep
            entry={refPreview.entry}
            loading={refPreview.loading}
            selectedSenseIds={selectedSenseIds}
            onToggleSense={toggleSense}
            notes={notes}
            onNotesChange={setNotes}
          />
        )}

        {step === 'preview' && refPreview.entry && (
          <DialogFooter>
            <Button variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button onClick={handleAdd} disabled={adding || selectedSenseIds.size === 0}>
              {adding ? 'Adding...' : `Add ${selectedSenseIds.size > 1 ? `(${selectedSenseIds.size} senses)` : 'word'}`}
            </Button>
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  )
}

// ─── Search Step ────────────────────────────────────────────────────────────

function SearchStep({
  query,
  onQueryChange,
  results,
  loading,
  onSelect,
}: {
  query: string
  onQueryChange: (q: string) => void
  results: RefEntry[]
  loading: boolean
  onSelect: (entry: RefEntry) => void
}) {
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    // Focus search input on mount
    setTimeout(() => inputRef.current?.focus(), 100)
  }, [])

  return (
    <div className="flex flex-col gap-3 min-h-0">
      {/* Search input */}
      <div className="flex items-center gap-2.5 px-3 py-2.5 border border-input rounded-md">
        <Search size={16} className="text-text-tertiary shrink-0" />
        <input
          ref={inputRef}
          type="text"
          value={query}
          onChange={(e) => onQueryChange(e.target.value)}
          placeholder="Type a word to search..."
          className="flex-1 bg-transparent border-none outline-none text-sm text-text-primary placeholder:text-muted-foreground"
        />
        {loading && (
          <span className="text-xs text-text-disabled animate-pulse">searching...</span>
        )}
      </div>

      {/* Results */}
      <div className="overflow-y-auto max-h-[50vh] -mx-1">
        {query.trim().length >= 2 && results.length === 0 && !loading && (
          <p className="text-sm text-text-tertiary text-center py-6">
            No words found for "{query}"
          </p>
        )}

        {results.map((entry) => {
          const primaryPos = entry.senses[0]?.partOfSpeech
          const translation = entry.senses[0]?.translations[0]?.text
          const ipa = entry.pronunciations[0]?.transcription
          const cleanIpa = ipa ? `/${ipa.replace(/^\/+|\/+$/g, '')}/` : undefined

          return (
            <button
              key={entry.id}
              type="button"
              onClick={() => onSelect(entry)}
              className="w-full text-left px-3 py-2.5 rounded-md hover:bg-surface-secondary transition-colors duration-150 flex items-baseline gap-3"
            >
              <span className="text-[15px] font-semibold text-text-primary">
                {entry.text}
              </span>
              {primaryPos && (
                <span className={`text-[11px] font-medium uppercase tracking-wide ${getPosColors(primaryPos).text}`}>
                  {POS_SHORT[primaryPos] ?? primaryPos.toLowerCase()}
                </span>
              )}
              {cleanIpa && (
                <span className="font-mono text-[12px] text-text-disabled">
                  {cleanIpa}
                </span>
              )}
              {translation && (
                <span className="text-[13px] text-text-secondary truncate">
                  {translation}
                </span>
              )}
            </button>
          )
        })}

        {query.trim().length < 2 && (
          <p className="text-sm text-text-tertiary text-center py-6">
            Start typing to search the dictionary
          </p>
        )}
      </div>
    </div>
  )
}

// ─── Preview Step ───────────────────────────────────────────────────────────

function PreviewStep({
  entry,
  loading,
  selectedSenseIds,
  onToggleSense,
  notes,
  onNotesChange,
}: {
  entry: RefEntry | null
  loading: boolean
  selectedSenseIds: Set<string>
  onToggleSense: (id: string) => void
  notes: string
  onNotesChange: (v: string) => void
}) {
  if (loading) {
    return (
      <div className="py-8 text-center">
        <span className="text-sm text-text-tertiary animate-pulse">Loading word details...</span>
      </div>
    )
  }

  if (!entry) {
    return (
      <div className="py-8 text-center">
        <span className="text-sm text-text-tertiary">Word not found in catalog</span>
      </div>
    )
  }

  const ipa = entry.pronunciations[0]?.transcription
  const cleanIpa = ipa ? `/${ipa.replace(/^\/+|\/+$/g, '')}/` : undefined

  return (
    <div className="flex flex-col gap-4 overflow-y-auto max-h-[55vh] pr-1">
      {/* Word header */}
      <div className="flex items-baseline gap-3">
        <span className="text-xl font-bold text-text-primary">{entry.text}</span>
        {cleanIpa && (
          <span className="font-mono text-sm text-text-tertiary">{cleanIpa}</span>
        )}
        {entry.cefrLevel && (
          <span className="text-[11px] font-semibold px-1.5 py-0.5 rounded bg-surface-secondary text-text-secondary uppercase">
            {entry.cefrLevel}
          </span>
        )}
      </div>

      {/* Senses */}
      <div className="flex flex-col gap-2">
        <p className="text-[11px] font-medium text-text-tertiary uppercase tracking-wider">
          Select definitions to add
        </p>
        {entry.senses.map((sense, idx) => (
          <SenseCard
            key={sense.id}
            sense={sense}
            index={idx + 1}
            selected={selectedSenseIds.has(sense.id)}
            onToggle={() => onToggleSense(sense.id)}
          />
        ))}
      </div>

      {/* Notes */}
      <div className="space-y-1.5">
        <label className="text-[11px] font-medium text-text-tertiary uppercase tracking-wider">
          Notes (optional)
        </label>
        <textarea
          value={notes}
          onChange={(e) => onNotesChange(e.target.value)}
          rows={2}
          placeholder="Add personal notes..."
          className="flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
        />
      </div>
    </div>
  )
}

// ─── Sense Card ─────────────────────────────────────────────────────────────

function SenseCard({
  sense,
  index,
  selected,
  onToggle,
}: {
  sense: RefSense
  index: number
  selected: boolean
  onToggle: () => void
}) {
  const pos = sense.partOfSpeech
  const colors = getPosColors(pos)

  return (
    <button
      type="button"
      onClick={onToggle}
      className={`text-left w-full rounded-lg border p-3 transition-all duration-150 ${
        selected
          ? 'border-cornflower bg-cornflower-light/50'
          : 'border-border-default hover:border-border-subtle bg-transparent'
      }`}
    >
      <div className="flex items-start gap-2.5">
        {/* Checkbox */}
        <div className={`mt-0.5 w-4 h-4 rounded border flex items-center justify-center shrink-0 transition-colors ${
          selected
            ? 'bg-cornflower border-cornflower text-text-on-accent'
            : 'border-border-default'
        }`}>
          {selected && <Check size={10} />}
        </div>

        <div className="flex-1 min-w-0">
          {/* POS + index */}
          <div className="flex items-center gap-2 mb-1">
            <span className="text-[11px] font-mono text-text-disabled">#{index}</span>
            {pos && (
              <span className={`text-[11px] font-medium uppercase tracking-wide ${colors.text}`}>
                {POS_SHORT[pos] ?? pos.toLowerCase()}
              </span>
            )}
          </div>

          {/* Definition */}
          {sense.definition && (
            <p className="text-sm text-text-primary leading-relaxed">{sense.definition}</p>
          )}

          {/* Translations */}
          {sense.translations.length > 0 && (
            <p className="text-[13px] text-text-secondary mt-1">
              {sense.translations.map(t => t.text).join(', ')}
            </p>
          )}

          {/* Examples */}
          {sense.examples.length > 0 && (
            <div className="mt-2 space-y-1">
              {sense.examples.slice(0, 2).map((ex) => (
                <div key={ex.id} className="text-[12.5px] text-text-tertiary italic">
                  "{ex.sentence}"
                  {ex.translation && (
                    <span className="not-italic text-text-disabled ml-1.5">
                      — {ex.translation}
                    </span>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </button>
  )
}
