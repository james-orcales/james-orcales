---
name: communication
description: >
  Load this before you write English in this repository. This includes replies, AGENTS.md,
  SPECIFICATION.md, doc comments, error strings, and commit messages. ASD-STE100 Issue 9
  Simplified Technical English gives 53 writing rules and 8 general recommendations. They control
  word choice, verb form, voice, sentence length, and punctuation. This is the house style, not a
  tool that you use only when the user asks.
---

# Simplified Technical English — ASD-STE100 Issue 9

ASD-STE100 is a controlled natural language. ASD (AeroSpace, Security and Defence Industries
Association of Europe) maintains it. It has 53 rules in 9 sections and a dictionary of 875
approved words and 1274 words that are not approved. Airlines made STE necessary because a
technician who reads an instruction incorrectly can die. Issue 9 identifies a second reader. It
says that controlled text is easier for "neural machine translation engines, and Large Language
Models (LLM)". The two readers cannot ask you a question.

**Write all your text and all documents in this style. Do not tell the user that you do it. Do
not give a before/after table unless the user asks for one.** For substitutions and approved
verbs, refer to `reference/dictionary.md`.

## Words

| Rule | Instruction |
| --- | --- |
| 1.1, 1.6 | Use an approved word, a technical noun, or a technical verb. Do not use other words. |
| 1.2, 9.2 | Use a word only in its approved part of speech and its approved meaning. The two are more narrow than in usual English. |
| 1.7, 1.13 | Do not use a technical noun as a verb, or a technical verb as a noun. Write "Apply oil to the valve", not "Oil the valve". |
| 1.11, 9.4 | Use one name for one item. Use one wording for one type of step. A different word tells the reader that the subject changed. |
| 9.3 | Do not use phrasal verbs. A verb with a preposition makes a meaning that the dictionary does not have. Write "extinguish the fire", not "put out the fire". |
| 9.1 | When a word-for-word replacement is not sufficient, use a different sentence construction. Do not use a replacement that breaks the grammar or changes the meaning. |
| 1.14 | Use American English spelling. |
| 1.8 | An organization can give approval to technical nouns that are not in the 875. In this repository these are Go identifiers, type and package names, and specified domain terms (`goroutine`, `mutex`, `struct`, `revision`, `commit`, `linter`). Rule 1.7 is still applicable — do not use them as verbs. |
| 1.9, 1.10 | Make a new technical noun short and easy to understand. Do not use slang or jargon words. |

## Verbs

| Rule | Instruction |
| --- | --- |
| 3.2 | Use only these six forms: the infinitive, the imperative, the simple present tense, the simple past tense, the simple future tense, and the past participle **as an adjective**. Do not use the present perfect ("has adjusted"), the past perfect ("had adjusted"), the progressive ("is adjusting"), or other complex constructions. |
| 3.4 | Do not use an auxiliary verb with a past participle. Identify the actor. "The volume control can be adjusted" becomes "You can adjust the volume control". The words `can`, `must`, and `will` are correct before a base verb. |
| 3.5 | Do not use an "-ing" form as a verb. It is permitted only in a technical noun (`Troubleshooting`, `welding torch`). Nine approved words have it: lighting, opening, routing, servicing, mating, missing, remaining, something, during. |
| 3.6 | Use the active voice. The passive voice is permitted only in descriptive text, and only if the agent is unknown. |
| 3.7 | Use a verb to describe an action, not a noun. Write "The meter shows 450 ohms", not "The meter gives an indication of 450 ohms". |

## Sentences and paragraphs

| Rule | Instruction |
| --- | --- |
| 5.1, 6.3 | Write a maximum of 20 words in a procedural sentence. Write a maximum of 25 words in a descriptive sentence. |
| 5.2 | Write only one instruction in each sentence. Two actions share a sentence only if they occur at the same time. |
| 5.3 | Write instructions in the imperative form. Add "must" only for safety or for an important condition. |
| 5.4 | Write the condition first, then a comma, then the command. "When the light comes on, set the switch to NORMAL." |
| 4.2 | Do not omit a noun, a verb, a subject, or an article to make a sentence shorter. Do not use contractions. Write "do not", not "don't". |
| 4.5 | Put an article or a demonstrative adjective before a noun. |
| 2.1, 2.2 | Write multi-word nouns of no more than three words. When a technical noun has more than three words, write it in full. |
| 4.3 | Use a vertical list for complex text. A colon starts the list. Each item starts with an uppercase letter. Do not nest the list. Do not use semicolons. Use a period only for a full sentence. |
| 4.4 | Connect related sentences with "and", "but", "then", "thus", or "as a result". |
| 6.1, 6.4, 6.5, 6.6 | Give information gradually. Write only one subject in each sentence. Write only one topic in each paragraph and start it with the topic sentence. Write a maximum of six sentences in each paragraph. The topic sentences together must give an outline of the document. |

## Safety and notes

| Rule | Instruction |
| --- | --- |
| 7.1 | A warning tells the reader that there is a risk of injury or death. A caution tells the reader that there is a risk of damage to objects. If there are the two levels of risk together, use a warning. |
| 7.2, 7.3 | Start with a clear and accurate command or condition. Then give the explanation of the risk or the possible result. |
| 5.5 | A note gives information only. It must not have an instruction, a requirement, a limit, a tolerance, or an imperative. To test this, remove all the notes. The procedure must still be correct. |

## Punctuation

| Rule | Instruction |
| --- | --- |
| 8.1 | The semicolon is the only punctuation mark that is not permitted. Write two sentences. |
| 8.2, 8.7 | Use hyphens to connect words that are directly related. A hyphenated group counts as one word. |
| 8.3 | Use parentheses for references, identifiers, work steps, abbreviations, singular and plural forms, short explanations, and alternatives. |
| 8.5, 8.6 | Count each of these as one word: a number, a number with its unit, an abbreviation, an alphanumeric identifier, quoted text, a title, a proper noun, and text in parentheses. |

## General recommendations

The standard says that these are not rules. They prevent the errors that writers make most.

| Item | Instruction |
| --- | --- |
| GR-1 | Use the conjunction "that" as much as possible. Write "Make sure that the valve is open". |
| GR-2 | The word "with" has three approved meanings. Read again each sentence that has it. Usually you must write the condition first. |
| GR-3, GR-4 | If a pronoun can refer to two nouns, use the noun again. Make sure that "this" refers to only one item. |
| GR-5 | Make sure that a word has its English meaning. A word that looks the same in a different language can have a different meaning. |
| GR-6 | Do not use Latin abbreviations. Write "for example" for "e.g.", "that is" for "i.e.", and "and so on" for "etc." Frequently you can remove them. |
| GR-7, GR-8 | Use gender-neutral language. The words "he" and "she" are not permitted. Use the possessive form only if you are sure that it is correct. |

STE is not for text where voice or style is the objective. Do not use it there. Do not omit a
safety condition, a scope qualifier, or a number to obey a word limit. Keep the longer sentence.
