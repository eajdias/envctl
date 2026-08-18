# win11-new (Windows 11 Environment Provisioner & State Replicator)

> Provisionador idempotente e determinístico em **Go** para replicação 1:1 de ambiente de desenvolvimento no Windows 11 PRO.

---

## 🎯 Objetivo
Transformar uma instalação limpa e recém-formatada do Windows 11 em uma estação de trabalho de desenvolvimento completa, configurada com shell MSYS2 otimizado, toolchains (Node/Volta, Python/uv/ruff, Go, .NET, Rust, Docker), Language Servers (16 LSPs), configs do OpenCode, terminal e as **56 Skills** de agentes, através de um único binário auto-contido.

---

## 🏛️ Arquitetura de Software (Clean Architecture)

O projeto é estruturado em camadas desacopladas seguindo os princípios de **Clean Architecture** e **SOLID**:

```
win11-new/
├── cmd/
│   └── win11-new/               # Entrypoint da aplicação (main.go, injeção de dependências)
├── internal/
│   ├── domain/                  # Camada de Domínio (Entidades e Interfaces/Contratos)
│   │   ├── entity/              # Entidades puras: Package, Toolchain, Skill, ConfigTemplate, AuditResult
│   │   └── repository/          # Interfaces: PackageInstaller, ShellExecutor, ConfigStore, ToolchainRunner
│   ├── usecase/                 # Casos de Uso da Aplicação
│   │   ├── provision_env.go     # Orquestrador completo de provisionamento
│   │   ├── verify_prereqs.go    # Checagem de privilégios de Admin, arquitetura, Windows build
│   │   ├── audit_system.go      # Auditoria do estado atual vs estado desejado (drift detection)
│   │   ├── sync_skills.go       # Extração e validação das 56 Skills
│   │   └── configure_shell.go   # Ajustes de MSYS2, nsswitch.conf, .bashrc, Windows Terminal
│   ├── infra/                   # Camada de Infraestrutura (Implementações concretas)
│   │   ├── winget/              # Adaptador para o Windows Package Manager (winget CLI)
│   │   ├── msys2/               # Adaptador para o MSYS2 (pacman, nsswitch.conf, chroot/paths)
│   │   ├── toolchain/           # Adaptadores para Volta, Dotnet, UV, Rustup, Go
│   │   ├── git/                 # Adaptador para Git config, SSH e GitHub CLI
│   │   ├── filesystem/          # Operações de I/O seguras, backup atômico e symlinks
│   │   └── embedded/            # Sistema de arquivos embutido no binário (//go:embed)
│   └── ui/                      # Interface com o Usuário
│       └── cli/                 # Comandos CLI, flags, saída com cores e barras de progresso
├── manifests/                   # Manifests declarativos de pacotes e ferramentas
├── configs/                     # Templates e arquivos de configuração fonte
└── docs/                        # Documentação técnica e ADRs (Architectural Decision Records)
```

---

## 💎 Princípios & Fundamentos
1. **Zero External Dependencies on Target:** O binário gerado roda nativamente no Windows 11 sem exigir Python, Node, PowerShell 7 ou Git pré-instalados.
2. **Idempotência Estrita:** Executar o utilitário 1 ou 100 vezes produz o mesmo estado final. Ferramentas já instaladas ou configuradas são detectadas e ignoradas com segurança.
3. **Self-Contained via `//go:embed`:** Todas as 56 Skills e arquivos de configuração residem dentro do próprio executável compilado.
4. **Segurança e Auditoria:** Nenhuma chave privada ou segredo é embutido no binário; chaves e credenciais seguem o padrão de injeção manual guiada pós-provisionamento.
