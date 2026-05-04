# Claude

The README.md file contains the product requirements

## Rules

1. There are multiple sources, each source should be implemented as a infrastructure
2. There are multiple notifications type, each type should be implemented as a infrastructure.
3. Follow the clean architecture.
4. Rules module should be extendable by adding files, or methods.
5. Write code as simple as you can.
6. After process data and logic, print the result message into console, stdout.

## Plans

Check each item when done

- [x] Design
  - [x] Find and suggest at least 2 services to get Weather information
  - [x] Design the configuration that can read configurations in files (JSON, YAML), or from environments, or from runtime arguments. Priorities follow: runtime arguments overrides environment variables, environments variables override the files, the configuration files override the default values configured in code.
- [ ] Implement
  - [x] Define configurations, variables for the application, and update README.md.
  - [x] Weather services infrastructure
  - [x] Main service to process the data and print to stdout.
  - [x] A notification module for multiple messenger apps.
  - [x] Implement Telegram integration.