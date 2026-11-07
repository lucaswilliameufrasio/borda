# Plano de desenvolvimento

Console local para a borda da rede. A pessoa abre e vê por onde a casa está saindo. O RouterOS continua sendo o lugar de firewall avançado, fila, VPN, usuário e script solto. Este repositório guarda o plano. Nenhum código de produto existe ainda.

## Resultado

A interface é um frontend separado. Ela fala com uma API em Go. Cada roteador tem um agent na LAN, e a API pode acompanhar vários agents ao mesmo tempo. O agent guarda a sessão daquele roteador e a credencial, só na memória. A interface mostra o caminho ativo, a prontidão da saída reserva e os aparelhos conhecidos de cada um. Toda escrita é uma ação nomeada, mostrada antes, e só segue depois de confirmação. Um modelo de linguagem pode explicar o estado ou propor uma ação dessa lista. Ele não recebe credencial, export, backup nem senha, e não grava configuração sozinho.

## Fora deste produto

- Reescrever o Winbox ou o WebFig. A tela de caminho pode ter um atalho para abrir o WebFig.
- Proxy livre de comando RouterOS. Texto livre não vira comando.
- Fila, VPN, usuário, firewall avançado e script solto.
- Banco de métricas dentro do roteador. O histórico fica no hub.
- Conta remota no modo local.

## Atores

| Ator | Pode | Não pode |
| --- | --- | --- |
| Morador | Ver se há saída, qual WAN está ativa e quais aparelhos estão na LAN. Pedir a prova da saída reserva e confirmar uma mudança de failover. | Gravar política sem a confirmação daquela ação. |
| Operador | O mesmo morador quando muda política: cadastrar uma WAN, mudar distância, desligar uma checagem. Confirma cada escrita. | Executar comando fora da lista fechada. |
| Agent | Único processo que guarda a sessão do roteador e a credencial, só na memória. | Enviar a senha ao hub, ao modelo ou ao log. |
| Hub | API, histórico e assistente. Acompanha um ou mais agents, um roteador por agent. | Guardar a senha do roteador. A interface não fala com o RouterOS. |
| Modelo | Ler um retrato já filtrado e devolver explicação ou proposta tipada. | Ter cliente do roteador ou cofre. |

Os dois modos de execução fazem parte do produto. No modo local, a API e os agents sobem nas máquinas da LAN e não há conta remota. No modo separado, cada agent fica na LAN do seu roteador e o hub pode estar em outro host. O hub guarda a identidade de cada agent e os retratos que eles enviam. Um comando do hub chega como pedido assinado. O agent só executa o que a política local permite, e uma escrita pede confirmação. A senha não atravessa esse canal.

## Jornadas

Cada jornada tem uma tela. Estado, cadastro e ação destrutiva não dividem o mesmo lugar.

### 1. Preparar a casa

Uma vez por roteador. O dia a dia não pede isso de novo. Incluir outro MikroTik repete esta jornada para aquele aparelho. A escolha fica nas configurações do tenant, fora da tela de caminho.

1. A pessoa informa o nome do tenant.
2. O hub lista os gateways das rotas desta máquina que aceitam SSH. Um só pede confirmação. Vários, a pessoa escolhe. Nenhum, ela digita o endereço.
3. A pessoa escolhe um cofre, vê o que vai digitar, e segue. A senha do roteador mora nesse cofre e não volta para a tela.
4. O hub lê a identidade do roteador e mostra a versão e o nome. A senha não entra nessa tela.
5. A pessoa cai no caminho ativo.

Cofres previstos: arquivo cifrado do próprio console, em que a pessoa cria a senha do cofre nessa tela; Ciphera; Infisical; chaveiro do sistema. Em todos os externos, o console lê o segredo na hora de usar e não copia o valor para o banco dele.

Vazio: nenhum gateway SSH. Falha: autenticação recusada ou roteador inalcançável, sem a senha na mensagem. Sucesso: identidade lida e caminho ativo aberto.

### 2. Ver o caminho ativo

A pessoa abre o console e vê qual rota padrão está ativa, se a checagem de alcance responde pela porta da WAN primária, e se a LAN conseguiria sair pela reserva USB. A reserva está pronta quando existem o NAT e o forward da LAN para a interface USB, nessa ordem, antes do descarte do resto.

Vazio: nenhuma rota padrão. Falha: agent sem sessão. Sucesso: um caminho nomeado e o tempo desde a última troca.

### 3. Provar a saída USB

Com o modem USB no cabo, dados móveis ligados e o Wi-Fi da casa desligado, a pessoa confirma a prova. O agent desabilita a rota primária, mede um destino público pela interface USB e pela LAN, e reabilita a rota. A casa fica sem a WAN primária por alguns segundos. Cancelar no meio reabilita a rota.

Sucesso: a medida volta e a rota primária está ativa de novo. Falha: a rota primária é reabilitada mesmo assim, e a tela diz que a prova não passou.

### 4. Cadastrar uma WAN

A pessoa escolhe uma porta que não é a porta da LAN, informa distância e nome, e confirma. O agent tira a porta da bridge, cria o NAT e o cliente DHCP nessa distância. Duas WANs não ficam com a mesma distância.

### 5. Ver aparelhos

A pessoa vê leases, vizinhos e o último anúncio. Abre um aparelho para ver endereço, nome e por qual caminho o tráfego dele saiu. Isso não altera firewall nem rota.

Estados do aparelho: visto agora, visto há pouco, ausente.

### 6. Perguntar ao assistente

A pessoa escreve o que está vendo. O modelo recebe só o retrato filtrado e pode propor uma ação da lista fechada. A proposta aparece para confirmar. Sem confirmação, nada é escrito. Se o retrato não bastar, a resposta diz o que falta ler. Ela não pede a senha.

O modelo vê nomes de interface, comentários, distâncias, flags, contagens e atrasos. Não vê senha, usuário, URL com credencial, conteúdo de usuário do roteador, export, backup, nem um script que carregue segredo.

### 7. Ligar um agent a um hub remoto

A pessoa aprova o agent na LAN. O hub passa a receber retrato e evento. A credencial continua no agent.

## Estados

- Caminho: cabo ativo, USB ativo, os dois inativos, prova em andamento.
- Política de failover: ordem das distâncias e checagem de cada WAN. A política desejada fica no hub. O RouterOS é a fonte do que está valendo agora.
- Sessão do agent: conectada, sem permissão de escrita, autenticada e recusada, roteador inalcançável.
- Troca de caminho é evento, com hora e motivo: a checagem falhou, a prova foi pedida, ou a rota foi reabilitada.

## Módulos

- Segredo. Um adaptador por cofre. O agent pede a senha na hora da sessão. Não há API para o resto do sistema ler o valor. Backup e export ficam no disco do agent, cifrados, e não entram no canal do hub nem no prompt.
- Sessão RouterOS. Operações nomeadas: ler rotas, ler forward, ler leases, desabilitar uma rota comentada, reabilitar, incluir uma WAN já desenhada. Timeout, erro legível, nenhuma senha na mensagem de erro.
- Política de caminho. Compara o desejado com o que o roteador tem. O endereço de checagem não pode ser gravado como DNS da LAN.
- Inventário. DHCP, ARP e neighbor, amarrados por MAC.
- Telemetria. Amostras de qual rota está ativa, perda e atraso da checagem, contadores das interfaces, presença de aparelho. A coleta para se o agent perder a sessão. Não inventa ponto.
- Assistente. Leitura sobre o retrato filtrado e uma lista fechada de propostas. A escrita passa pelo comando, com confirmação.
- Comando. Idempotência por comentário estável. Recusa comando fora da lista. Mostra o que vai mudar antes de aplicar.

Os comentários estáveis são o contrato das escritas. A prova só pode desabilitar a rota padrão cujo comentário é `borda-primary`. Sem esse comentário, com mais de uma rota assim, ou sem USB em pé, forward da LAN para essa USB e um endereço de LAN, a prova não começa e nada é desabilitado. Uma WAN cadastrada usa o comentário `borda-wan-` seguido do nome em minúsculas, no cliente DHCP e no NAT. A prova continua só podendo desabilitar `borda-primary`. A leitura da fatia 1 não depende desses comentários: ela mostra a rota padrão que o roteador já tem.

## Observabilidade

Separada do cadastro.

- O caminho: qual rota padrão está ativa, se a checagem responde pela WAN primária, se a interface USB tem endereço, se a sessão USB está de pé, e se o forward da LAN para o USB existe.
- A casa: leases ativos, aparelho novo, aparelho que sumiu, nome anunciado.
- O evento: troca de WAN, prova iniciada, prova concluída, agent sem sessão.

Um gráfico de vazão da interface pode entrar depois do evento de troca. Sem o evento, o gráfico não explica a queda. O gráfico de CPU e de porta que o RouterOS já publica não é esta tela.

## Fatias

Cada fatia fecha uma jornada e fica demonstrável sozinha. A fatia 1 tem API e frontend neste repositório. A fatia 2 também. As duas foram exercitadas num CHR RouterOS 7.25.1 de laboratório, com SSH só em 127.0.0.1. A leitura mostrou a identidade `lab`, a rota padrão ativa por `ether1` e a checagem `1.1.1.1` respondendo. A prova confirmada respondeu 412 `PROOF_NOT_READY` com motivo `missing_primary`: esse CHR não tem LTE e a rota do DHCP não carrega o comentário `borda-primary`. A rota permaneceu ativa. AC-08 não fecha neste CHR, porque não há USB para medir. Rota ausente, sessão falha e dois agents seguem verificados só no roteador falso. A fatia 3 lista os aparelhos da leitura atual, juntando DHCP, ARP e neighbor pelo MAC. No mesmo CHR, o único aparelho foi o gateway visto por ARP em `ether1`, sem nome anunciado e sem lease. Abrir esse aparelho mostrou o endereço e o caminho ativo, e a rota padrão continuou ativa. Neighbor e lease nesse CHR estavam vazios; a junção deles ficou no teste. Não há amostra guardada, então visto há pouco e ausente continuam de fora. A fatia 4 grava a troca de caminho no hub, com hora e um destes motivos: a checagem falhou, a prova foi pedida, ou a rota foi reabilitada. A primeira leitura conectada só marca o caminho e não cria evento. Troca com a checagem ainda respondendo fica só como observação, sem motivo inventado. Prova recusada não entra. Se a reabilitação falha, fica só o pedido da prova. Listar o histórico não consulta o roteador. A troca de caminho fica até o roteador sair do cadastro. No CHR, com o cofre fechado, a tela de trocas abriu vazia: só o cadastro e a lista de eventos, sem retrato e sem escrita. A rota padrão do DHCP continuou ativa, com `disabled=false` e comentário vazio. Os três motivos ficaram no teste, porque esse CHR não trocou de caminho e a prova não começa. AC-08 continua aberto. A fatia 5 responde a partir do retrato filtrado e só pode propor a prova da USB. Senha, usuário, URL com credencial, export, backup e conteúdo de usuário ficam fora desse retrato. Proposta fora da lista não é executada, e sem confirmação a prova não começa. O assistente responde pelas regras e não chama um modelo. No CHR, com o cofre fechado, pedir um export foi recusado e perguntar o caminho respondeu que falta a sessão. Não houve chamada de ação. A rota padrão do DHCP continuou ativa, com `disabled=false` e comentário vazio. A confirmação da prova e o filtro do retrato ficaram no teste. A fatia 6 aprova o agent e o hub passa a receber o retrato filtrado e os eventos. A senha não vai junto. Um comando de escrita enviado ao hub não roda: a confirmação continua no agent. O endereço do hub remoto fica sem escolha no modo local. No CHR, com o cofre fechado, a aprovação mostrou a sessão fechada, zero rotas e nenhuma troca, sem pedir senha. Não houve comando de escrita. A rota padrão do DHCP continuou ativa, com `disabled=false` e comentário vazio. O retrato conectado e a recusa do comando ficaram no teste. AC-08 continua aberto.

### 1. Setup e caminho ativo, só leitura

A pessoa escolhe o cofre e o agent daquele roteador. A API em Go entrega o retrato. O frontend, separado, mostra a lista de roteadores e, ao abrir um, o caminho ativo, as rotas, o forward e os leases. A credencial fica fora do código, fora do log e fora do frontend.

| ID | Resultado observável |
| --- | --- |
| AC-01 | Depois do setup de um roteador, a pessoa abre o console, escolhe esse roteador e vê a rota padrão ativa, a porta de saída e se a checagem responde. A senha não aparece na tela nem no log. |
| AC-02 | Sem rota padrão, a tela daquele roteador mostra o estado vazio e não inventa um caminho. |
| AC-03 | Sem sessão com um roteador, a tela mostra a falha daquele agent e não mostra segredo. Os outros agents continuam visíveis. |
| AC-04 | Leases aparecem quando o DHCP daquele roteador os tem. A lista vazia é um estado válido. |
| AC-05 | Esta fatia não grava no roteador. |
| AC-06 | Dois agents cadastrados aparecem como dois roteadores. O retrato de um não se mistura com o do outro. |

### 2. Prova do USB

Confirmação, desabilitar e reabilitar a rota primária, e a medida de um destino público.

| ID | Resultado observável |
| --- | --- |
| AC-07 | Sem confirmação, a rota primária permanece ativa. |
| AC-08 | Com confirmação, a prova mede o destino pela USB e pela LAN, e a rota primária termina ativa. |
| AC-09 | Cancelar ou falhar a prova também deixa a rota primária ativa. |

### 3. Inventário de aparelhos

A pessoa abre um aparelho da leitura atual e vê endereço, nome anunciado e o caminho ativo daquele roteador. DHCP, ARP e neighbor com o mesmo MAC são um aparelho só. Nada é alterado no firewall nem na rota. Sem amostra guardada, a tela não mostra visto há pouco nem ausente.

| ID | Resultado observável |
| --- | --- |
| AC-10 | A pessoa abre um aparelho da leitura atual e vê o endereço, o nome anunciado quando existe, e o caminho ativo do roteador. A senha não aparece. |
| AC-11 | Abrir o aparelho não altera o firewall nem a rota. |
| AC-12 | DHCP, ARP e neighbor com o mesmo MAC viram um aparelho. Um MAC inválido não vira aparelho. |

### 4. Histórico de troca de caminho

Cada troca registrada no hub tem hora e motivo. O roteador não é consultado como arquivo desse histórico. A tela abre com o cofre fechado. A primeira leitura conectada não vira evento. A troca fica guardada até o roteador sair do cadastro, então o hub não apaga evento.

| ID | Resultado observável |
| --- | --- |
| AC-13 | Uma troca guardada no hub tem hora e um motivo da lista: checagem falhou, prova pedida ou rota reabilitada. A lista não lê o roteador e não mostra senha. |
| AC-14 | Abrir o histórico não altera o firewall nem a rota. |
| AC-15 | Leitura inicial, prova que não começou e troca com a checagem respondendo não criam motivo. Falha ao reabilitar não registra a rota como reabilitada. |

### 5. Assistente em modo proposta

A pessoa pergunta o que está vendo. A resposta sai do retrato filtrado: interfaces, comentários, distâncias, flags e contagens. Senha, usuário, URL com credencial, export, backup e conteúdo de usuário não entram. A única proposta da lista é a prova da USB. Fora da lista, nada é executado. Sem confirmação, a prova não começa. Esta fatia não chama um modelo. A resposta fica nas regras do retrato.

| ID | Resultado observável |
| --- | --- |
| AC-16 | O retrato usado na resposta não contém senha, usuário, URL com credencial, export, backup nem conteúdo de usuário. |
| AC-17 | Uma proposta fora da lista não é executada. |
| AC-18 | Sem confirmação, a prova não começa. Com confirmação, só a prova da lista pode seguir. |

### 6. Agent separado e hub remoto

A pessoa aprova o agent. O hub passa a receber o retrato filtrado e os eventos. A senha e o usuário ficam no agent. Um comando de escrita enviado ao hub não é executado: a confirmação continua no agent. O endereço do hub remoto fica sem escolha no modo local. Esta fatia guarda a aprovação neste console.

| ID | Resultado observável |
| --- | --- |
| AC-19 | Depois da aprovação, o hub mostra o retrato e os eventos. A senha e o usuário não estão nesse retrato. |
| AC-20 | Sem aprovação, o hub não recebe o retrato. |
| AC-21 | Um comando de escrita enviado ao hub não altera o roteador. A confirmação continua no agent. |

### 7. Cadastrar uma WAN

A pessoa informa nome, porta e distância, vê o comentário `borda-wan-<nome>` e confirma. O agent tira a porta da bridge, cria o NAT masquerade e o cliente DHCP com esse comentário. Sem confirmação, nada muda. Porta com endereço, porta da rota ativa, distância já usada ou o mesmo comentário em outra porta recusam e não escrevem. Repetir o mesmo cadastro não cria outro objeto. A rota `borda-primary` não é desabilitada. No CHR, com o cofre fechado, a confirmação respondeu que o cofre não abriu. A rota padrão do DHCP continuou ativa, com `disabled=false` e comentário vazio, e não nasceu cliente DHCP nem NAT com comentário de WAN. O cadastro em si ficou no teste.

| ID | Resultado observável |
| --- | --- |
| AC-22 | Sem confirmação, nada é escrito. |
| AC-23 | Com confirmação, o cliente DHCP e o NAT nascem com `borda-wan-<nome>`. A rota `borda-primary` não é desabilitada. |
| AC-24 | Porta ocupada, distância ocupada ou comentário em outra porta recusam e não escrevem. |
| AC-25 | Repetir o mesmo nome, porta e distância não cria um segundo objeto. |

## Ordem e verificação

A jornada de laboratório fecha a leitura da fatia 1 neste CHR e a recusa da prova quando a rota comentada ou a USB não existem. Ela não fecha a medida USB. Um teste automatizado cobre o contrato de cada módulo com um roteador falso. Teste verde sem essa jornada não substitui o que o CHR mostrou.

O script de leitura precisou de um ajuste encontrado nessa jornada. O RouterOS executa o payload SSH linha a linha, então cada consulta fica numa linha só. No RouterOS 7, o ping com `count=1` só devolve `received` quando o script pede `proplist=received`.

## Decisões fechadas

- A fatia 1 é uma API em Go e um frontend SvelteKit separado. A interface nasce em inglês e português do Brasil, com Paraglide. Um hub acompanha vários agents. Cada agent fala com um MikroTik.
- No setup, a pessoa escolhe o cofre. Os quatro adaptadores fazem parte dessa escolha: arquivo cifrado, Ciphera, Infisical e chaveiro do sistema.
- A leitura funciona em qualquer roteador que tenha rota padrão. Ela não exige os comentários estáveis. Esses comentários só entram na prova e no cadastro de WAN.
- O endereço de checagem é configuração do tenant e nunca vira DNS da LAN. O campo nasce vazio. A tela mostra `1.1.1.1` como sugestão. O valor só é gravado quando a pessoa preenche. Hoje esse valor fica no cadastro do roteador.
- O endereço do hub remoto permanece sem escolha enquanto o console for local. A senha fica no agent. Um host só entra na conta quando um agent estiver na LAN de um roteador e a pessoa for olhar o retrato a partir de outro host.
- O assistente responde pelas regras do retrato filtrado. Nenhum modelo é chamado. Se um modelo entrar depois, fica atrás do módulo de assistente, sem cliente do roteador. Ele vê o retrato filtrado e a pergunta, e só pode devolver texto ou a proposta `proof`. O provedor previsto para esse momento é SpaceXAI.
- A troca de caminho fica guardada até o roteador sair do cadastro. A presença de aparelho guarda a transição, não cada leitura. Visto agora é a leitura corrente. Visto há pouco é a última aparição dentro de 1 hora. Ausente é mais antigo que 1 hora. O registro do aparelho sai depois de 7 dias sem aparecer. A coleta para quando o agent perde a sessão. Essa coleta ainda não está na tela.
- A licença do repositório é MIT, no arquivo `LICENSE`.
- Uma WAN cadastrada leva o comentário `borda-wan-` mais o nome em minúsculas. O nome começa com letra e usa letras, números e hífen. O mesmo comentário vai no cliente DHCP e no NAT masquerade. A prova da USB continua só podendo desabilitar `borda-primary`.

## Decisões em aberto

- A medida real da prova USB. Ela precisa de um roteador com LTE no cabo e do comentário `borda-primary`. O CHR de laboratório não fecha a AC-08.

## O que este plano não autoriza

Esta jornada não autoriza o roteador da casa. As decisões em aberto não bloqueiam a leitura.
