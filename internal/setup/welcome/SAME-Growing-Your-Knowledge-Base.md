---
title: "SAME — Growing Your Knowledge Base"
tags: [same, reference, knowledge-management]
content_type: hub
---

# Growing Your Knowledge Base

The more relevant notes you have, the smarter your AI becomes. Here's how to build a useful knowledge base.

## What To Add

### High-Value Content

| Type | Why It Helps |
|------|--------------|
| **Architecture decisions** | AI understands your system design |
| **API documentation** | AI can reference endpoints, schemas |
| **Coding standards** | AI follows your conventions |
| **Project READMEs** | AI understands project context |
| **Meeting notes** | AI knows what was discussed/decided |
| **Troubleshooting guides** | AI can help debug faster |

### Your Own Content

The best knowledge base is YOUR content:
- Blog posts you've written
- Documentation you've created
- Notes from your learning
- Project documentation
- Internal wikis (if you have rights)

## Safe Sources

### Always OK
- Content you created
- Your company's internal docs (with permission)
- Open source documentation
- Public API references
- Your own blog/writing
- Notes from books you're reading
- Course notes (for personal use)

### Check First
- Paywalled content (respect subscriptions)
- Copyrighted materials (fair use varies)
- Third-party documentation (usually OK for reference)

### Avoid
- Scraping sites that prohibit it (check ToS)
- Storing others' copyrighted work as your own
- Personal data without consent
- Proprietary/confidential information you don't own

## Practical Tips

### 1. Start With Decisions

Every time you make a technical decision, document it:

```markdown
---
title: "Decision: Use Tailwind CSS"
tags: [frontend, styling, decision]
content_type: decision
---

# Decision: Use Tailwind CSS

## Context
Need a styling approach for the new dashboard.

## Decision
Use Tailwind CSS with the default config.

## Rationale
- Utility-first fits our component approach
- Good DX with IDE autocomplete
- Team already knows it
```

### 2. Document As You Go

When you solve a problem, write it down:
- What was the issue?
- What did you try?
- What worked?

Future you (and your AI) will thank you.

### 3. Import Existing Docs

If you have docs in other formats:
- Copy/paste relevant sections into markdown
- Use tools to convert (Notion export, Google Docs → MD)
- Keep the structure (headings, lists)

### 4. Create Hub Notes

For each major topic, create a "hub" that links to related notes:

```markdown
---
title: "Authentication Hub"
tags: [auth, hub]
content_type: hub
---

# Authentication

Central reference for auth-related decisions.

## Decisions
- [[Decision: Use JWT]]
- [[Decision: Session Duration]]

## Implementation
- [[Auth Flow Diagram]]
- [[Token Refresh Logic]]
```

## Web Content

If you want to save web content for reference:

### Recommended Approach
1. **Summarize, don't copy** — Write your own notes about what you learned
2. **Quote sparingly** — Brief quotes with attribution are generally OK
3. **Link to sources** — Include URLs for reference
4. **Focus on facts** — Technical facts are more reusable than opinion pieces

### Example: Learning From Docs

Instead of copying React docs, write:

```markdown
---
title: "React Hooks Notes"
tags: [react, hooks, learning]
---

# React Hooks Notes

My notes from learning React hooks.

## Key Concepts

- `useState` returns [value, setter] tuple
- `useEffect` runs after render, cleanup on unmount
- Custom hooks must start with "use"

## Gotchas I Encountered

- Don't call hooks conditionally
- useEffect dependencies matter (learned the hard way)

## Resources
- https://react.dev/reference/react/hooks
```

This is YOUR knowledge, informed by the docs — not a copy.

## Legal Note

You are responsible for ensuring you have the right to use content you add to your knowledge base. When in doubt:
- Create original content
- Summarize rather than copy
- Link to sources
- Check terms of service
- Respect copyright

SAME is a local tool — what you store is your responsibility.
