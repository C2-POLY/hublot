// SPDX-License-Identifier: AGPL-3.0-or-later
// Tiny Markdown subset for issue-form help text: paragraphs, "- " lists, **bold**,
// `code` and [links](https://…). Built from React elements, never raw HTML.
import type { ReactNode } from 'react'

function inline(text: string, keyBase: string): ReactNode[] {
  const out: ReactNode[] = []
  const re = /\*\*([^*]+)\*\*|`([^`]+)`|\[([^\]]+)\]\((https?:\/\/[^\s)]+)\)/g
  let last = 0
  let m: RegExpExecArray | null
  let k = 0
  while ((m = re.exec(text))) {
    if (m.index > last) out.push(text.slice(last, m.index))
    const key = `${keyBase}-${k++}`
    if (m[1] !== undefined) out.push(<strong key={key}>{m[1]}</strong>)
    else if (m[2] !== undefined) out.push(<code key={key} className="font-mono text-[0.92em]">{m[2]}</code>)
    else out.push(<a key={key} href={m[4]} target="_blank" rel="noopener noreferrer">{m[3]}</a>)
    last = re.lastIndex
  }
  if (last < text.length) out.push(text.slice(last))
  return out
}

export function Markdown({ source }: { source: string }) {
  const blocks = source.replace(/\r\n/g, '\n').trim().split(/\n{2,}/)
  return (
    <div className="flex flex-col gap-2 rounded-xl border-franc-dashed bg-notice px-4 py-3 text-[15px] leading-normal">
      {blocks.map((b, i) => {
        const lines = b.split('\n')
        if (lines.every((l) => /^\s*[-*] /.test(l))) {
          return (
            <ul key={i} className="m-0 list-disc pl-5">
              {lines.map((l, j) => (
                <li key={j}>{inline(l.replace(/^\s*[-*] /, ''), `${i}-${j}`)}</li>
              ))}
            </ul>
          )
        }
        const text = lines.map((l) => l.replace(/^#+\s*/, '')).join(' ')
        return (
          <p key={i} className="m-0">
            {inline(text, String(i))}
          </p>
        )
      })}
    </div>
  )
}
