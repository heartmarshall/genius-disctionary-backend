import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuCheckboxItem,
} from '@/components/ui/dropdown-menu'
import { useWordDetail } from '@/hooks/useWordDetail'
import { useWordEdit } from '@/hooks/useWordEdit'
import { useDictionary } from '@/hooks/useDictionary'
import type { PartOfSpeech } from '@/types/dictionary'

const PART_OF_SPEECH_VALUES: PartOfSpeech[] = [
  'NOUN', 'VERB', 'ADJECTIVE', 'ADVERB', 'PRONOUN',
  'PREPOSITION', 'CONJUNCTION', 'INTERJECTION', 'PHRASE', 'IDIOM', 'OTHER',
]

interface SenseForm {
  definition: string
  partOfSpeech: string
  translations: string[]
  examples: { sentence: string; translation: string }[]
}

interface ImageForm {
  url: string
  caption: string
}

interface PronunciationForm {
  transcription: string
  audioUrl: string
  region: string
}

interface WordEditDialogProps {
  wordId: string | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function WordEditDialog({ wordId, open, onOpenChange }: WordEditDialogProps) {
  const { t } = useTranslation('dictionary')
  const { entry } = useWordDetail(wordId ?? undefined)
  const { updateWord, loading: saving } = useWordEdit()
  const { topics: allTopics } = useDictionary({})

  const [text, setText] = useState('')
  const [notes, setNotes] = useState('')
  const [topicIds, setTopicIds] = useState<string[]>([])
  const [senses, setSenses] = useState<SenseForm[]>([])
  const [images, setImages] = useState<ImageForm[]>([])
  const [pronunciations, setPronunciations] = useState<PronunciationForm[]>([])

  // Initialize form from entry
  useEffect(() => {
    if (!entry) return
    setText(entry.text)
    setNotes(entry.notes ?? '')
    setTopicIds(entry.topics.map((t) => t.id))
    setSenses(
      entry.senses.map((s) => ({
        definition: s.definition ?? '',
        partOfSpeech: s.partOfSpeech ?? '',
        translations: s.translations.map((tr) => tr.text),
        examples: s.examples.map((ex) => ({
          sentence: ex.sentence,
          translation: ex.translation ?? '',
        })),
      }))
    )
    setImages(
      entry.images.map((img) => ({
        url: img.url,
        caption: img.caption ?? '',
      }))
    )
    setPronunciations(
      entry.pronunciations.map((p) => ({
        transcription: p.transcription,
        audioUrl: p.audioUrl ?? '',
        region: p.region ?? '',
      }))
    )
  }, [entry])

  const handleSave = async () => {
    if (!wordId) return
    try {
      await updateWord(wordId, {
        text,
        notes: notes || undefined,
        topicIDs: topicIds,
        senses: senses.map((s) => ({
          definition: s.definition || undefined,
          partOfSpeech: s.partOfSpeech || undefined,
          translations: s.translations
            .filter((t) => t.trim())
            .map((t) => ({ text: t })),
          examples: s.examples
            .filter((e) => e.sentence.trim())
            .map((e) => ({
              sentence: e.sentence,
              translation: e.translation || undefined,
            })),
        })),
        images: images
          .filter((img) => img.url.trim())
          .map((img) => ({
            url: img.url,
            caption: img.caption || undefined,
          })),
        pronunciations: pronunciations
          .filter((p) => p.transcription.trim())
          .map((p) => ({
            transcription: p.transcription,
            audioUrl: p.audioUrl || undefined,
            region: p.region || undefined,
          })),
      })
      onOpenChange(false)
    } catch {
      toast.error(t('error.saveFailed'))
    }
  }

  // Sense helpers
  const addSense = () =>
    setSenses([...senses, { definition: '', partOfSpeech: '', translations: [''], examples: [] }])

  const removeSense = (index: number) =>
    setSenses(senses.filter((_, i) => i !== index))

  const updateSense = (index: number, field: keyof SenseForm, value: unknown) =>
    setSenses(senses.map((s, i) => (i === index ? { ...s, [field]: value } : s)))

  // Topic toggle
  const toggleTopic = (id: string) =>
    setTopicIds((prev) =>
      prev.includes(id) ? prev.filter((t) => t !== id) : [...prev, id]
    )

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{t('edit.title')}</DialogTitle>
          <DialogDescription className="sr-only">
            {t('edit.title')}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {/* Word text */}
          <div className="space-y-2">
            <label className="text-sm font-medium">{t('edit.word')}</label>
            <Input value={text} onChange={(e) => setText(e.target.value)} />
          </div>

          {/* Topics */}
          <div className="space-y-2">
            <label className="text-sm font-medium">{t('edit.topics')}</label>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm" className="w-full justify-start">
                  {topicIds.length > 0
                    ? allTopics
                        .filter((t) => topicIds.includes(t.id))
                        .map((t) => t.name)
                        .join(', ')
                    : t('edit.topics')}
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent className="w-56">
                {allTopics.map((topic) => (
                  <DropdownMenuCheckboxItem
                    key={topic.id}
                    checked={topicIds.includes(topic.id)}
                    onCheckedChange={() => toggleTopic(topic.id)}
                  >
                    {topic.name}
                  </DropdownMenuCheckboxItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>

          {/* Notes */}
          <div className="space-y-2">
            <label className="text-sm font-medium">{t('edit.notes')}</label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={3}
              className="flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
            />
          </div>

          <Separator />

          {/* Senses */}
          <div className="space-y-4">
            <label className="text-sm font-medium">{t('edit.senses')}</label>
            {senses.map((sense, senseIdx) => (
              <div key={senseIdx} className="space-y-3 p-3 border rounded-md relative">
                <Button
                  variant="ghost"
                  size="icon"
                  className="absolute top-2 right-2 h-7 w-7 text-muted-foreground hover:text-poppy"
                  onClick={() => removeSense(senseIdx)}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>

                {/* Part of speech */}
                <div className="space-y-1">
                  <label className="text-xs text-muted-foreground">{t('edit.partOfSpeech')}</label>
                  <div className="flex gap-1 flex-wrap">
                    {PART_OF_SPEECH_VALUES.map((pos) => (
                      <Badge
                        key={pos}
                        variant={sense.partOfSpeech === pos ? 'default' : 'outline'}
                        className="cursor-pointer text-xs"
                        onClick={() => updateSense(senseIdx, 'partOfSpeech', sense.partOfSpeech === pos ? '' : pos)}
                      >
                        {t(`pos.${pos}`)}
                      </Badge>
                    ))}
                  </div>
                </div>

                {/* Definition */}
                <div className="space-y-1">
                  <label className="text-xs text-muted-foreground">{t('edit.definition')}</label>
                  <Input
                    value={sense.definition}
                    onChange={(e) => updateSense(senseIdx, 'definition', e.target.value)}
                  />
                </div>

                {/* Translations */}
                <div className="space-y-1">
                  <label className="text-xs text-muted-foreground">{t('edit.translations')}</label>
                  {sense.translations.map((tr, trIdx) => (
                    <div key={trIdx} className="flex gap-2">
                      <Input
                        value={tr}
                        onChange={(e) => {
                          const updated = [...sense.translations]
                          updated[trIdx] = e.target.value
                          updateSense(senseIdx, 'translations', updated)
                        }}
                      />
                      {sense.translations.length > 1 && (
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-9 w-9 shrink-0"
                          onClick={() => {
                            const updated = sense.translations.filter((_, i) => i !== trIdx)
                            updateSense(senseIdx, 'translations', updated)
                          }}
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      )}
                    </div>
                  ))}
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-xs"
                    onClick={() => updateSense(senseIdx, 'translations', [...sense.translations, ''])}
                  >
                    <Plus className="h-3 w-3 mr-1" />
                    {t('edit.addTranslation')}
                  </Button>
                </div>

                {/* Examples */}
                <div className="space-y-1">
                  <label className="text-xs text-muted-foreground">{t('edit.examples')}</label>
                  {sense.examples.map((ex, exIdx) => (
                    <div key={exIdx} className="flex gap-2 items-start">
                      <div className="flex-1 space-y-1">
                        <Input
                          placeholder={t('edit.sentence')}
                          value={ex.sentence}
                          onChange={(e) => {
                            const updated = [...sense.examples]
                            updated[exIdx] = { ...ex, sentence: e.target.value }
                            updateSense(senseIdx, 'examples', updated)
                          }}
                        />
                        <Input
                          placeholder={t('edit.translation')}
                          value={ex.translation}
                          onChange={(e) => {
                            const updated = [...sense.examples]
                            updated[exIdx] = { ...ex, translation: e.target.value }
                            updateSense(senseIdx, 'examples', updated)
                          }}
                        />
                      </div>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-9 w-9 shrink-0 mt-0"
                        onClick={() => {
                          const updated = sense.examples.filter((_, i) => i !== exIdx)
                          updateSense(senseIdx, 'examples', updated)
                        }}
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  ))}
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-xs"
                    onClick={() =>
                      updateSense(senseIdx, 'examples', [
                        ...sense.examples,
                        { sentence: '', translation: '' },
                      ])
                    }
                  >
                    <Plus className="h-3 w-3 mr-1" />
                    {t('edit.addExample')}
                  </Button>
                </div>
              </div>
            ))}
            <Button variant="outline" size="sm" onClick={addSense}>
              <Plus className="h-4 w-4 mr-1" />
              {t('edit.addSense')}
            </Button>
          </div>

          <Separator />

          {/* Media — Images */}
          <div className="space-y-3">
            <label className="text-sm font-medium">{t('edit.images')}</label>
            {images.map((img, idx) => (
              <div key={idx} className="flex gap-2 items-start">
                <div className="flex-1 space-y-1">
                  <Input
                    placeholder={t('edit.imageUrl')}
                    value={img.url}
                    onChange={(e) => {
                      const updated = [...images]
                      updated[idx] = { ...img, url: e.target.value }
                      setImages(updated)
                    }}
                  />
                  <Input
                    placeholder={t('edit.caption')}
                    value={img.caption}
                    onChange={(e) => {
                      const updated = [...images]
                      updated[idx] = { ...img, caption: e.target.value }
                      setImages(updated)
                    }}
                  />
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-9 w-9 shrink-0"
                  onClick={() => setImages(images.filter((_, i) => i !== idx))}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </div>
            ))}
            <Button
              variant="ghost"
              size="sm"
              className="text-xs"
              onClick={() => setImages([...images, { url: '', caption: '' }])}
            >
              <Plus className="h-3 w-3 mr-1" />
              {t('edit.addImage')}
            </Button>
          </div>

          {/* Media — Pronunciations */}
          <div className="space-y-3">
            <label className="text-sm font-medium">{t('edit.pronunciations')}</label>
            {pronunciations.map((pron, idx) => (
              <div key={idx} className="flex gap-2 items-start">
                <div className="flex-1 space-y-1">
                  <Input
                    placeholder={t('edit.transcription')}
                    value={pron.transcription}
                    onChange={(e) => {
                      const updated = [...pronunciations]
                      updated[idx] = { ...pron, transcription: e.target.value }
                      setPronunciations(updated)
                    }}
                  />
                  <Input
                    placeholder={t('edit.audioUrl')}
                    value={pron.audioUrl}
                    onChange={(e) => {
                      const updated = [...pronunciations]
                      updated[idx] = { ...pron, audioUrl: e.target.value }
                      setPronunciations(updated)
                    }}
                  />
                  <Input
                    placeholder={t('edit.region')}
                    value={pron.region}
                    onChange={(e) => {
                      const updated = [...pronunciations]
                      updated[idx] = { ...pron, region: e.target.value }
                      setPronunciations(updated)
                    }}
                  />
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-9 w-9 shrink-0"
                  onClick={() => setPronunciations(pronunciations.filter((_, i) => i !== idx))}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </div>
            ))}
            <Button
              variant="ghost"
              size="sm"
              className="text-xs"
              onClick={() =>
                setPronunciations([...pronunciations, { transcription: '', audioUrl: '', region: '' }])
              }
            >
              <Plus className="h-3 w-3 mr-1" />
              {t('edit.addPronunciation')}
            </Button>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('edit.cancel')}
          </Button>
          <Button onClick={handleSave} disabled={saving}>
            {saving ? t('edit.saving') : t('edit.save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
