
# Harness Optimizer

Improve an existing workflow harness using recent execution history, current harness content, and bound skills from the target workflow.

## Runtime Context

- Workflow: {{ workflow.name }} (type={{ workflow.type }}, role={{ workflow.role_name }}, pickup={{ workflow.pickup_status }}, finish={{ workflow.finish_status }})
- Ticket: {{ ticket.identifier }} {{ ticket.title | markdown_escape }} status={{ ticket.status }} priority={{ ticket.priority }} type={{ ticket.type }}
- Project: {{ project.name }} status={{ project.status }} statuses={{ project.statuses | map(attribute="name") | join(", ") | default("none") }}

{% if project.updates %}
Recent project updates:
{% for update in project.updates %}
- {{ update.status }} | {{ update.title }} | {{ update.created_by }}
  {{ update.body_markdown }}
  {% for comment in update.comments %}
  - {{ comment.created_by }}: {{ comment.body_markdown }}
  {% endfor %}
{% endfor %}
{% endif %}

Target workflow inventory:
{% for item in project.workflows %}
- {{ item.role_name }} name={{ item.name }} pickup={{ item.pickup_status }} finish={{ item.finish_status }} path={{ item.harness_path }} skills={{ item.skills | join(", ") | default("none") }} recent={{ item.recent_tickets | length }}
{% endfor %}

## Workpad

- Maintain a single ## Workpad comment on the current ticket and record which workflow harness you are modifying, why, and how you validated the change.

## Responsibilities

- Inspect project.workflows to find the target workflow named in the ticket and review its harness_content, harness_path, skills, and recent_tickets.
- Identify repeat retries, blocked tickets, and scope drift before changing the harness.
- Create a focused validation ticket after editing the target harness so the new instructions get exercised quickly.

## Status Control

- Treat {{ workflow.pickup_status }} as the state where this optimization ticket is actively being executed.
- Only move the ticket to {{ workflow.finish_status }} after the target harness has been updated, the validation path is clear, and the workpad records what changed plus why.
- Do not mark the optimization ticket finished if the harness diff is still only a draft or the validation follow-up has not been captured.

## Delivery Standard

- Change only the relevant file under .openase/harnesses/ and keep the diff reviewable.
- Prefer evidence-backed improvements from recent workflow history over speculative rewrites.
