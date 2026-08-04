# Rule: Integridade de release

## Garantias obrigatórias

- Fragmentos pendentes são candidatos, nunca prova de uma release.
- Uma publicação de tag consome apenas notas e fragmentos preparados e versionados.
- Tags, publicação e verificação são fatos distintos com evidências tipadas e confinadas; a verificação vincula a publicação exata e os digests dos assets.
- Manifestos de release, notas e fragmentos arquivados tornam-se imutáveis após o corte.
- Política ausente após adoção, fragmentos duplicados, snapshots desatualizados, drift de tag ou estados mais fortes que a evidência bloqueiam declarações estritas de release.
- Automação de release não pode utilizar staging amplo, sobrescrita de tag ou force-pushing.
- Backfill histórico relata confiança e lacunas sem fabricar fatos.

## Gates mínimos

Execute `pose release check --version vX.Y.Z --strict`, checagens de compatibilidade e validação completa de módulos antes de criar a tag. Exija que `pose release status` projeta `verified` antes de declarar a publicação como concluída.
