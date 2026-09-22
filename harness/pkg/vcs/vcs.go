// Package vcs judges one commit subject against the rules a branch's history obeys: its size,
// its conventional shape, and the fixup and merge commits a branch squashes or rebases away before
// it lands. It runs no git. The caller reads the history and hands over each subject, and the
// judge states the one rule the commit breaks.
package vcs

import "local/james-orcales/shared/std/sim/aver/default"

// SUBJECT_SIZE_MAXIMUM is the widest subject a commit carries: a review tool cuts one near 72 to
// 100 characters, and a wider one scrolls.
const SUBJECT_SIZE_MAXIMUM = 100

// SUBJECT_TEXT_SIZE_MAXIMUM bounds a subject the judge reads. A subject is untrusted, and one past
// this bound is past the size cap whatever it holds, thus the caller states it without the judge.
const SUBJECT_TEXT_SIZE_MAXIMUM = 4096

// SUBJECT_TEXT_SIZE_MINIMUM is one byte, since a commit of no subject states nothing to judge.
const SUBJECT_TEXT_SIZE_MINIMUM = 1

// Subject is the first line of one commit message, as the history hands it over.
type Subject string

// Subject_Invariants bounds the subject by what the judge reads.
func Subject_Invariants(value Subject, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    SUBJECT_TEXT_SIZE_MINIMUM,
    SUBJECT_TEXT_SIZE_MAXIMUM,
  ).
    Ensure()
}

// Sized_Subject is a subject inside the size cap, which every rule past the cap reads.
type Sized_Subject string

// Sized_Subject_Invariants bounds the subject by the size cap.
func Sized_Subject_Invariants(value Sized_Subject, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), SUBJECT_TEXT_SIZE_MINIMUM, SUBJECT_SIZE_MAXIMUM).
    Ensure()
}

// Merged records that the commit has more than one parent.
type Merged bool

// Merged_Invariants proves both kinds of commit occur.
func Merged_Invariants(value Merged, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Sometimes(bool(value), "A commit is a merge.").
    Ensure()
}

// Answer is one answer a rule gives.
type Answer bool

// Answer_Invariants proves both answers occur.
func Answer_Invariants(value Answer, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Sometimes(bool(value), "The rule holds for the subject.").
    Ensure()
}

// VERDICT_NONE is a commit that breaks no rule.
const VERDICT_NONE Verdict = 0

// VERDICT_SIZE is a subject past the size cap.
const VERDICT_SIZE Verdict = 1

// VERDICT_CONVENTIONAL is a subject off the conventional shape.
const VERDICT_CONVENTIONAL Verdict = 2

// VERDICT_FIXUP is a fixup commit, which autosquashes into its target.
const VERDICT_FIXUP Verdict = 3

// VERDICT_MERGE is a merge commit, which a rebase takes away.
const VERDICT_MERGE Verdict = 4

// Verdict is the one rule a commit breaks, or none.
type Verdict int

// Verdict_Invariants admits each verdict.
func Verdict_Invariants(value Verdict, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(int(value), int(VERDICT_NONE), int(VERDICT_MERGE)).
    Ensure()
}

// Merge_Verdict is the rule one merge commit breaks: its size, the merge itself, or none.
type Merge_Verdict Verdict

// Merge_Verdict_Invariants admits the three verdicts a merge draws.
func Merge_Verdict_Invariants(value Merge_Verdict, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Int(
    int(value),
    int(VERDICT_NONE),
    int(VERDICT_SIZE),
    int(VERDICT_MERGE),
  ).
    Ensure()
}

// Pull_Subject is a subject that opens as GitHub's pull request merge does.
type Pull_Subject string

// Pull_Subject_Invariants bounds the subject by its opening and the judge's bound.
func Pull_Subject_Invariants(value Pull_Subject, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    PULL_SUBJECT_SIZE_MINIMUM,
    SUBJECT_TEXT_SIZE_MAXIMUM,
  ).
    Ensure()
}

// PULL_SUBJECT_SIZE_MINIMUM is the opening a pull request merge subject holds.
const PULL_SUBJECT_SIZE_MINIMUM = len(PULL_MERGE_OPENING)

// Sha_Subject is a subject that opens as GitHub's test merge does.
type Sha_Subject string

// Sha_Subject_Invariants bounds the subject by its opening and the judge's bound.
func Sha_Subject_Invariants(value Sha_Subject, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    SHA_SUBJECT_SIZE_MINIMUM,
    SUBJECT_TEXT_SIZE_MAXIMUM,
  ).
    Ensure()
}

// SHA_SUBJECT_SIZE_MINIMUM is the opening a test merge subject holds.
const SHA_SUBJECT_SIZE_MINIMUM = len(SHA_MERGE_OPENING)

// Subtree_Subject is a merge subject that opens as git subtree's do.
type Subtree_Subject string

// Subtree_Subject_Invariants bounds the subject by the shorter opening and the size cap.
func Subtree_Subject_Invariants(value Subtree_Subject, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(
    len(value),
    SUBTREE_SUBJECT_SIZE_MINIMUM,
    SUBJECT_SIZE_MAXIMUM,
  ).
    Ensure()
}

// SUBTREE_SUBJECT_SIZE_MINIMUM is the shorter opening a subtree merge subject holds.
const SUBTREE_SUBJECT_SIZE_MINIMUM = len(SUBTREE_ADD_OPENING)

// Judge states the rule one commit breaks. A size past the cap is stated before the shape, since
// the one fix a writer makes for it rewrites the shape too.
func Judge(subject Subject, merged Merged) (verdict Verdict) {
  defer func() { Verdict_Invariants(verdict, "judge.verdict") }()
  Subject_Invariants(subject, "judge.subject")
  Merged_Invariants(merged, "judge.merged")
  if merged {
    return Verdict(judge_merge(subject))
  }
  // A synthetic merge lands among the ordinary commits on a shallow checkout, and Git's own
  // revert wraps a subject it did not write, thus neither is judged.
  if synthetic(subject) {
    return VERDICT_NONE
  }
  if reverted(subject) {
    return VERDICT_NONE
  }
  if len(subject) > SUBJECT_SIZE_MAXIMUM {
    return VERDICT_SIZE
  }
  if fixup(Sized_Subject(subject)) {
    return VERDICT_FIXUP
  }
  if !conventional(Sized_Subject(subject)) {
    return VERDICT_CONVENTIONAL
  }
  return VERDICT_NONE
}

// States the rule one merge commit breaks. A merge git subtree writes vendors another history,
// which no rebase can, and a synthetic merge is a host's own.
func judge_merge(subject Subject) (verdict Merge_Verdict) {
  defer func() { Merge_Verdict_Invariants(verdict, "judge_merge.verdict") }()
  Subject_Invariants(subject, "judge_merge.subject")
  if len(subject) > SUBJECT_SIZE_MAXIMUM {
    return Merge_Verdict(VERDICT_SIZE)
  }
  if subtree(Sized_Subject(subject)) {
    return Merge_Verdict(VERDICT_NONE)
  }
  if synthetic(subject) {
    return Merge_Verdict(VERDICT_NONE)
  }
  return Merge_Verdict(VERDICT_MERGE)
}

// PULL_MERGE_OPENING opens the subject of a pull request merge GitHub writes.
const PULL_MERGE_OPENING = "Merge pull request #"

// PULL_MERGE_SOURCE stands between the pull request's number and the branch it merged.
const PULL_MERGE_SOURCE = " from "

// SHA_MERGE_OPENING opens the subject of the merge GitHub writes to test a pull request.
const SHA_MERGE_OPENING = "Merge "

// SHA_MERGE_TARGET stands between the two object names of that merge.
const SHA_MERGE_TARGET = " into "

// SHA_SIZE_MINIMUM is the shortest object name GitHub writes into a merge subject.
const SHA_SIZE_MINIMUM = 7

// SHA_SIZE_MAXIMUM is the longest object name Git writes, a SHA-256 name in hex.
const SHA_SIZE_MAXIMUM = 64

// Reports whether GitHub wrote the subject for a merge of its own: a pull request's, or the one
// it writes to test a pull request against its base.
func synthetic(subject Subject) (yes Answer) {
  defer func() { Answer_Invariants(yes, "synthetic.yes") }()
  Subject_Invariants(subject, "synthetic.subject")
  text := string(subject)
  if len(text) > len(PULL_MERGE_OPENING)-1 {
    if text[:len(PULL_MERGE_OPENING)] == PULL_MERGE_OPENING {
      return pull_merge(Pull_Subject(subject))
    }
  }
  if len(text) > len(SHA_MERGE_OPENING)-1 {
    if text[:len(SHA_MERGE_OPENING)] == SHA_MERGE_OPENING {
      return sha_merge(Sha_Subject(subject))
    }
  }
  return false
}

// Reports whether a pull request merge subject names its number in digits, then the branch it
// merged.
func pull_merge(subject Pull_Subject) (yes Answer) {
  defer func() { Answer_Invariants(yes, "pull_merge.yes") }()
  Pull_Subject_Invariants(subject, "pull_merge.subject")
  text := string(subject)
  count := len(PULL_MERGE_OPENING)
  for count < len(text) {
    if text[count] < '0' {
      break
    }
    if text[count] > '9' {
      break
    }
    count++
  }
  if count == len(PULL_MERGE_OPENING) {
    return false
  }
  rest := text[count:]
  if len(rest) < len(PULL_MERGE_SOURCE)+1 {
    return false
  }
  if rest[:len(PULL_MERGE_SOURCE)] != PULL_MERGE_SOURCE {
    return false
  }
  return !blank(Byte(rest[len(PULL_MERGE_SOURCE)]))
}

// Reports whether a merge subject names two object names and nothing past them.
func sha_merge(subject Sha_Subject) (yes Answer) {
  defer func() { Answer_Invariants(yes, "sha_merge.yes") }()
  Sha_Subject_Invariants(subject, "sha_merge.subject")
  text := string(subject)
  // Two runs of lower case hex digits, each one object name at most: the first stands before
  // the target, and the second closes the subject.
  runs := [SHA_RUN_COUNT]int{}
  count := len(SHA_MERGE_OPENING)
  for run := range SHA_RUN_COUNT {
    start := count
    for range SHA_SIZE_MAXIMUM {
      if count == len(text) {
        break
      }
      if !bool(hex_digit(Byte(text[count]))) {
        break
      }
      count++
    }
    runs[run] = count - start
    if run > 0 {
      break
    }
    if len(text)-count < len(SHA_MERGE_TARGET) {
      return false
    }
    if text[count:count+len(SHA_MERGE_TARGET)] != SHA_MERGE_TARGET {
      return false
    }
    count += len(SHA_MERGE_TARGET)
  }
  if count != len(text) {
    return false
  }
  if runs[0] < SHA_SIZE_MINIMUM {
    return false
  }
  return Answer(runs[1] > SHA_SIZE_MINIMUM-1)
}

// SHA_RUN_COUNT is the two object names a test merge names.
const SHA_RUN_COUNT = 2

// BYTE_MINIMUM is the least byte a subject holds: git writes no zero byte into one.
const BYTE_MINIMUM = 1

// BYTE_MAXIMUM is the last value one byte holds.
const BYTE_MAXIMUM = 255

// Byte is one byte of a subject.
type Byte byte

// Byte_Invariants admits every byte a subject holds.
func Byte_Invariants(value Byte, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Uint8(uint8(value), BYTE_MINIMUM, BYTE_MAXIMUM).
    Ensure()
}

// Reports whether one byte is a lower case hex digit.
func hex_digit(value Byte) (yes Answer) {
  defer func() { Answer_Invariants(yes, "hex_digit.yes") }()
  Byte_Invariants(value, "hex_digit.value")
  if value > '0'-1 {
    if value < '9'+1 {
      return true
    }
  }
  if value > 'a'-1 {
    if value < 'f'+1 {
      return true
    }
  }
  return false
}

// Reports whether one byte is the space a regular expression's \s names.
func blank(value Byte) (yes Answer) {
  defer func() { Answer_Invariants(yes, "blank.yes") }()
  Byte_Invariants(value, "blank.value")
  switch value {
  case ' ', '\t', '\n', '\f', '\r':
    return true
  }
  return false
}

// REVERT_OPENING opens the subject Git writes for a revert, before the subject it reverts.
const REVERT_OPENING = "Revert \""

// Reports whether Git wrote the subject for a revert: the opening, a subject of one byte at
// least, and the closing quote the subject ends on.
func reverted(subject Subject) (yes Answer) {
  defer func() { Answer_Invariants(yes, "reverted.yes") }()
  Subject_Invariants(subject, "reverted.subject")
  text := string(subject)
  if len(text) < len(REVERT_OPENING)+len("a\"") {
    return false
  }
  if text[:len(REVERT_OPENING)] != REVERT_OPENING {
    return false
  }
  return Answer(text[len(text)-1] == '"')
}

// SUBTREE_ADD_OPENING opens the merge subject the subtree add command writes.
const SUBTREE_ADD_OPENING = "Add '"

// SUBTREE_PULL_OPENING opens the merge subject the subtree pull command writes.
const SUBTREE_PULL_OPENING = "Merge commit '"

// Reports whether git subtree wrote the merge subject. A subtree merge worded by hand is no
// different from any other merge, thus it is none.
func subtree(subject Sized_Subject) (yes Answer) {
  defer func() { Answer_Invariants(yes, "subtree.yes") }()
  Sized_Subject_Invariants(subject, "subtree.subject")
  text := string(subject)
  if len(text) > len(SUBTREE_ADD_OPENING)-1 {
    if text[:len(SUBTREE_ADD_OPENING)] == SUBTREE_ADD_OPENING {
      return holds(Subtree_Subject(subject), "' from commit '")
    }
  }
  if len(text) > len(SUBTREE_PULL_OPENING)-1 {
    if text[:len(SUBTREE_PULL_OPENING)] == SUBTREE_PULL_OPENING {
      return holds(Subtree_Subject(subject), "' as '")
    }
  }
  return false
}

// Fragment is the text one subtree subject holds between the names it states.
type Fragment string

// Fragment_Invariants bounds the fragment by the two git subtree writes.
func Fragment_Invariants(value Fragment, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), FRAGMENT_SIZE_MINIMUM, FRAGMENT_SIZE_MAXIMUM).
    Ensure()
}

// FRAGMENT_SIZE_MINIMUM is the fragment the subtree pull subject holds.
const FRAGMENT_SIZE_MINIMUM = len("' as '")

// FRAGMENT_SIZE_MAXIMUM is the fragment the subtree add subject holds.
const FRAGMENT_SIZE_MAXIMUM = len("' from commit '")

// Reports whether the subject holds the fragment, byte for byte.
func holds(subject Subtree_Subject, fragment Fragment) (yes Answer) {
  defer func() { Answer_Invariants(yes, "holds.yes") }()
  Subtree_Subject_Invariants(subject, "holds.subject")
  Fragment_Invariants(fragment, "holds.fragment")
  text := string(subject)
  for start := 0; start+len(fragment) < len(text)+1; start++ {
    if text[start:start+len(fragment)] == string(fragment) {
      return true
    }
  }
  return false
}

// FIXUP_OPENING opens the subject a fixup commit carries.
const FIXUP_OPENING = "fixup!"

// SQUASH_OPENING opens the subject a squash commit carries.
const SQUASH_OPENING = "squash!"

// Reports whether the subject belongs squashed into an earlier commit: one Git wrote to be
// autosquashed, or one that answers a review rather than stating a change. A review answer names
// the review, what it did, and what it answered, thus a subject that merely names a review stands.
func fixup(subject Sized_Subject) (yes Answer) {
  defer func() { Answer_Invariants(yes, "fixup.yes") }()
  Sized_Subject_Invariants(subject, "fixup.subject")
  text := string(subject)
  for _, opening := range [...]string{FIXUP_OPENING, SQUASH_OPENING} {
    if len(text) > len(opening)-1 {
      if text[:len(opening)] == opening {
        return true
      }
    }
  }
  acted, answered := false, false
  for _, action := range [...]Word{"address", "apply"} {
    if mentions(subject, action) {
      acted = true
    }
  }
  for _, target := range [...]Word{"comment", "feedback", "nit"} {
    if mentions(subject, target) {
      answered = true
    }
  }
  if mentions(subject, "review") {
    if acted {
      if answered {
        return true
      }
    }
  }
  phrases := [...]Word{"cr comment", "code review comment", "review fix", "review nit"}
  for _, phrase := range phrases {
    if mentions(subject, phrase) {
      return true
    }
  }
  return false
}

// WORD_SIZE_MINIMUM is the shortest word a review answer names.
const WORD_SIZE_MINIMUM = len("nit")

// WORD_SIZE_MAXIMUM is the longest phrase a review answer names.
const WORD_SIZE_MAXIMUM = len("code review comment")

// Word is one lower case word or phrase a review answer names.
type Word string

// Word_Invariants bounds the word by the shortest and the longest a review answer names.
func Word_Invariants(value Word, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int(len(value), WORD_SIZE_MINIMUM, WORD_SIZE_MAXIMUM).
    Ensure()
}

// Reports whether the subject holds the lower case word, in any case of its ASCII letters.
func mentions(subject Sized_Subject, word Word) (yes Answer) {
  defer func() { Answer_Invariants(yes, "mentions.yes") }()
  Sized_Subject_Invariants(subject, "mentions.subject")
  Word_Invariants(word, "mentions.word")
  text := string(subject)
  for start := 0; start+len(word) < len(text)+1; start++ {
    matched := true
    for index := range len(word) {
      character := text[start+index]
      if character > 'A'-1 {
        if character < 'Z'+1 {
          character += 'a' - 'A'
        }
      }
      if character != word[index] {
        matched = false
        break
      }
    }
    if matched {
      return true
    }
  }
  return false
}

// Reports whether the subject is conventional: a lower case type, an optional scope in
// parentheses, an optional mark of a breaking change, a colon and a space, and a description that
// opens on a byte no space is.
func conventional(subject Sized_Subject) (yes Answer) {
  defer func() { Answer_Invariants(yes, "conventional.yes") }()
  Sized_Subject_Invariants(subject, "conventional.subject")
  text := string(subject)
  offset := 0
  for offset < len(text) {
    if text[offset] < 'a' {
      break
    }
    if text[offset] > 'z' {
      break
    }
    offset++
  }
  if offset == 0 {
    return false
  }
  if offset < len(text) {
    if text[offset] == '(' {
      end := offset + 1
      for end < len(text) {
        if text[end] == ')' {
          break
        }
        end++
      }
      if end == offset+1 {
        return false
      }
      if end == len(text) {
        return false
      }
      offset = end + 1
    }
  }
  if offset < len(text) {
    if text[offset] == '!' {
      offset++
    }
  }
  if len(text)-offset < len(": a") {
    return false
  }
  if text[offset:offset+len(": ")] != ": " {
    return false
  }
  return !blank(Byte(text[offset+len(": ")]))
}
