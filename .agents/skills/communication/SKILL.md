---
name: communication
description: >
  Load BEFORE writing any English in this repo — chat replies, AGENTS.md, SPECIFICATION.md, doc
  comments, error strings, commit messages. ASD-STE100 Issue 9 Simplified Technical English: the
  53 writing rules and 8 general recommendations that fix word choice, verb form, voice, sentence
  length, and punctuation. The house voice, not a tool you run when asked.
---

# Simplified Technical English — ASD-STE100 Issue 9

A controlled natural language from ASD (AeroSpace, Security and Defence Industries Association of
Europe): 53 rules in 9 sections, plus 875 approved and 1274 non-approved words. Airlines asked for
it because a technician who misreads an instruction can die. Issue 9 names a second reader —
"neural machine translation engines, and Large Language Models (LLM)". Neither can ask what you
meant.

**Write every reply and every document this way. Do not announce it. Do not emit a before/after
table unless asked.** Substitutions and approved verbs: `reference/dictionary.md`.

## Words

- **1.1, 1.6** Use an approved word, a technical noun, or a technical verb. Nothing else.
- **1.2, 9.2** Use a word only in its approved part of speech and approved meaning. Both are
  narrower than ordinary English.
- **1.7, 1.13** Never use a technical noun as a verb, or a technical verb as a noun. Write "Apply
  oil to the valve", not "Oil the valve".
- **1.11, 9.4** One item, one name. One kind of step, one wording. A synonym tells the reader that
  the subject changed.
- **9.3** No phrasal verbs — verb plus preposition builds a meaning the dictionary does not carry.
  Write "extinguish the fire" and "release fumes", not "put out" and "give off".
- **9.1** When no word-for-word swap works, change the sentence construction. Do not force a
  substitution that breaks the grammar or shifts the meaning. **1.14** American English spelling.

Rule 1.8 lets an organization approve technical nouns past the 875. Here that set is Go
identifiers, type and package names, and settled domain terms (`goroutine`, `mutex`, `struct`,
`revision`, `commit`, `linter`). Rule 1.7 still binds — do not verb them. Keep a coined term short
and plain (1.9). No slang, no jargon (1.10).

## Verbs

- **3.2** Six forms only: infinitive, imperative, simple present, simple past, simple future, past
  participle **as an adjective**. Banned: present perfect ("has adjusted"), past perfect ("had
  adjusted"), progressive ("is adjusting"), every other compound construction.
- **3.4** No auxiliary plus past participle. Name the actor. "The volume control can be adjusted"
  becomes "You can adjust the volume control". "The temperature must be adjusted" becomes "Adjust
  the temperature". `can`, `must`, and `will` are correct before a base verb.
- **3.5** No "-ing" as a verb. It is permitted only inside a technical noun (`Troubleshooting`,
  `welding torch`). Nine approved words carry it: lighting, opening, routing, servicing, mating,
  missing, remaining, something, during.
- **3.6** Active voice. Passive is permitted only in descriptive text, and only if the agent is
  unknown.
- **3.7** Name an action with a verb, not a noun. "The meter shows 450 ohms", not "The meter gives
  an indication of 450 ohms".

## Sentences and paragraphs

- **5.1** Procedural sentence: 20 words maximum. **6.3** Descriptive sentence: 25 words maximum.
- **5.2** One instruction per sentence. Two actions share a sentence only if they occur together.
- **5.3** Instructions take the imperative. Add "must" only for safety or an important condition.
- **5.4** Condition first, then a comma, then the command: "When the light comes on, set the switch
  to NORMAL."
- **4.2** Do not drop a noun, verb, subject, or article to save space, and do not contract — write
  "do not", never "don't". **4.5** Put an article or a demonstrative before a noun.
- **2.1** A multi-word noun holds three words maximum. **2.2** Write a longer one in full first.
- **4.3** Put complex material in a vertical list: colon lead-in, each item starts uppercase, no
  nesting, no semicolons, a period only on full sentences.
- **4.4** Join related sentences with "and", "but", "then", "thus", "as a result".
- **6.1, 6.4–6.6** Give information gradually. One subject per sentence. One topic per paragraph,
  opened by its topic sentence, six sentences maximum, together outlining the document.

## Safety and notes

- **7.1** A **warning** covers risk of injury or death. A **caution** covers risk of damage to
  objects. When both apply, use a warning.
- **7.2, 7.3** Command or condition first, then the explanation of the risk.
- **5.5** A note gives information only — no instruction, requirement, limit, tolerance, or
  imperative. Test: delete every note and the procedure must still work.

## Punctuation

- **8.1** The semicolon is the only banned mark. Write two sentences.
- **8.2, 8.7** Hyphenate directly related words. A hyphenated group counts as one word.
- **8.3** Parentheses hold references, identifiers, work steps, abbreviations, singular/plural,
  short explanations, and alternatives.
- **8.5, 8.6** One word each: a number, a number with its unit, an abbreviation, an alphanumeric
  identifier, quoted text, a title, a proper noun, a parenthetical.

## General recommendations

- **GR-1** Keep "that": "Make sure **that** the valve is open."
- **GR-2** "with" carries three approved meanings. Reread every sentence that uses it. The usual
  fix is to write the condition first.
- **GR-3, GR-4** If a pronoun could bind to two nouns, repeat the noun. Give "this" one referent.
- **GR-6** No Latin abbreviations: "for example" not "e.g.", "that is" not "i.e.", "and so on" not
  "etc." Frequently you can delete them.
- **GR-7, GR-8** Use gender-neutral language — no "he" or "she". Use the possessive only if sure.

STE is deliberately flat — do not apply it where voice or persuasion is the point. Never drop a
safety condition, a scope qualifier, or a number to meet a word limit. Keep the longer sentence.
