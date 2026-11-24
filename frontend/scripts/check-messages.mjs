// Checks the translations in messages/ against messages/en.json: every
// locale has exactly the keys of en.json, and each message has the same
// {arguments} and <tags> as its English one. Exits 1 on a problem.
import fs from 'fs'
import path from 'path'
import { fileURLToPath, pathToFileURL } from 'url'

// flatten turns nested messages into { 'a.b.c': 'text' }
export function flatten(messages, prefix = '', out = {}) {
  for (const [key, value] of Object.entries(messages)) {
    const name = prefix ? `${prefix}.${key}` : key
    if (value && typeof value === 'object') flatten(value, name, out)
    else out[name] = value
  }
  return out
}

// placeholders returns the argument names and tag names of an ICU message,
// e.g. "{count, plural, one {# <b>item</b>} other {...}}" -> count, b.
// Throws on a message ICU can't read (an unbalanced brace).
export function placeholders(message) {
  const args = new Set()
  const tags = new Set()
  let i = 0
  const fail = (why) => {
    throw new Error(`${why} at ${i}`)
  }
  const take = (pattern) => {
    const match = pattern.exec(message.slice(i))
    if (match) i += match[0].length
    return match
  }

  const text = (nested) => {
    while (i < message.length) {
      const c = message[i]
      if (c === "'" && message[i + 1] === "'") {
        i += 2
      } else if (c === "'" && '{}#<|'.includes(message[i + 1] ?? '')) {
        // Quoted literal text up to the next apostrophe
        const end = message.indexOf("'", i + 1)
        i = end < 0 ? message.length : end + 1
      } else if (c === '{') {
        i++
        argument()
      } else if (c === '}') {
        if (nested) return
        fail('unexpected }')
      } else {
        const tag = c === '<' && take(/^<\/?([A-Za-z][\w-]*)\s*\/?>/)
        if (tag) tags.add(tag[1])
        else i++
      }
    }
    if (nested) fail('missing }')
  }

  const argument = () => {
    const name = take(/^\s*([^\s,{}]+)\s*/)
    if (!name) fail('argument without a name')
    args.add(name[1])
    if (take(/^\}/)) return
    const type = take(/^,\s*(\w+)\s*/)
    if (!type) fail('expected , or }')
    if (take(/^\}/)) return
    if (!take(/^,/)) fail('expected , or }')
    if (['plural', 'select', 'selectordinal'].includes(type[1])) {
      for (;;) {
        take(/^\s*(offset:\s*\d+\s*)?/)
        if (take(/^\s*\}/)) return
        if (!take(/^[^\s{}]+\s*\{/)) fail('expected an option')
        text(true)
        i++
      }
    }
    // A number or date style: skip to its closing brace
    for (let depth = 1; depth > 0; i++) {
      if (i >= message.length) fail('missing }')
      if (message[i] === '{') depth++
      if (message[i] === '}') depth--
    }
  }

  text(false)
  return { args: [...args].sort().join(','), tags: [...tags].sort().join(',') }
}

// checkLocale lists what is wrong with one locale's messages
export function checkLocale(enMessages, localeMessages) {
  const en = flatten(enMessages)
  const other = flatten(localeMessages)
  const problems = []
  for (const [key, english] of Object.entries(en)) {
    if (!(key in other)) {
      problems.push(`missing key ${key}`)
      continue
    }
    if (typeof other[key] !== 'string' || other[key] === '') {
      problems.push(`${key} is not a text`)
      continue
    }
    try {
      const want = placeholders(english)
      const got = placeholders(other[key])
      if (want.args !== got.args) {
        problems.push(`${key} has arguments {${got.args}}, en.json {${want.args}}`)
      }
      if (want.tags !== got.tags) {
        problems.push(`${key} has tags <${got.tags}>, en.json <${want.tags}>`)
      }
    } catch (error) {
      problems.push(`${key} is not a valid message: ${error.message}`)
    }
  }
  for (const key of Object.keys(other)) {
    if (!(key in en)) problems.push(`extra key ${key}`)
  }
  return problems
}

function main() {
  const dir = path.join(path.dirname(fileURLToPath(import.meta.url)), '..', 'messages')
  const read = (file) => JSON.parse(fs.readFileSync(path.join(dir, file), 'utf8'))
  const en = read('en.json')
  let failed = false
  // en.json against itself catches English messages ICU can't read
  for (const file of fs
    .readdirSync(dir)
    .filter((f) => f.endsWith('.json'))
    .sort()) {
    const problems = checkLocale(en, read(file))
    for (const problem of problems) console.error(`${file}: ${problem}`)
    failed ||= problems.length > 0
  }
  if (failed) process.exit(1)
  console.log('messages: every locale has the keys and placeholders of en.json')
}

if (import.meta.url === pathToFileURL(process.argv[1]).href) main()
