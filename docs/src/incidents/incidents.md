# Incident management
Practice incident management skills so they become second nature, so you don't struggle to follow the process.

## Implementing an incident management process
The steps for implementing an incident management process can be summarized as follows:
1. Define exit criteria for types of incidents
1. Avoid groupthink and formalize assessment of operations with a risk assessment matrix. Risk assessment could be combined with Analytic Hierarchy Process (AHP).
1. Appoint delegates for critical functions to avoid single points of failure
1. Agree organization wise on effort required for different levels of severity and priority
1. Define and automate response plans, and make sure communication section includes backup communication methods
1. Work with developers to create playbooks for all services, which need verification and approval process

## Incident declaration automation
`devopsbot` automates incident declaration:
- Incident responders and commanders should focus on the essential tasks of resolving the incident
- Automate the repetitive steps in the incident management process
- Keep affected stakeholders informed about the status of the incident
- Standardize and document the procedure to lay the foundation for something to be improved upon

An incident cannot be planned for, the nature of it is that it just happens.
Incidents happen all the time for everyone, it is an unfortunate but natural part of life.
That does not mean they can be taken for granted. Teams should have a process
for managing them and learn from them.

An incident is defined as something that:
- Negatively affects customers
- Was not planned for
- Cannot be resolved within 1 hour

### During incidents
The workflow during an incident is as follows:

![incident declaration flow using the bot](./devopsbot.drawio.png)

### After incidents
When an incident has been declared as resolved, there is a need to communicate the resolution and
learn from the experience.

The workflow after an incident is suggested as follows:

![after](./after-incident.drawio.png)
