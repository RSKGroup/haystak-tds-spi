# Protocol specs

Microsoft Open Specifications documents for the protocols we implement, kept here as
offline, version-locked copies. They aren't our work; the TDS wire is built from them,
mostly MS-TDS. Captured 2026-06-09.

| File | Spec | What it's for | Version captured |
|---|---|---|---|
| `MS-TDS.pdf` | [MS-TDS] Tabular Data Stream Protocol | The core wire spec: framing, pre-login, LOGIN7, token streams, TYPE_INFO, RPC/`sp_*`, attention. The primary reference for M3. | 41.0 — 2026-03-30 (232 pp) |
| `MC-SQLR.pdf` | [MC-SQLR] SQL Server Resolution Protocol | UDP 1434 instance resolution (SQL Server Browser): how clients discover named instances. Needed only for named-instance support. | current as of 2026-06-09 (30 pp) |
| `MS-BINXML.pdf` | [MS-BINXML] SQL Server Binary XML Structure | Binary encoding of the `xml` data type. Relevant only once the XML type is supported, well past read-only v1. | 9.0 — 2025-10-31 (31 pp) |

## Where to find newer versions

Landing pages (subscribe or check for newer revisions here):
- [MS-TDS] — <https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-tds/b46a581a-39de-4745-b076-ec4dbb7d13ec>
- [MC-SQLR] — <https://learn.microsoft.com/en-us/openspecs/windows_protocols/mc-sqlr/1f9c44f8-26df-4c3d-9b73-2a18f76d2f31>
- [MS-BINXML] — <https://learn.microsoft.com/en-us/openspecs/sql_server_protocols/ms-binxml/11ab6e8d-2472-44d1-a9e6-bddf000e12f6>

Direct PDF hosts (Azure Front Door; `[`/`]` are URL-encoded as `%5b`/`%5d`):
- Windows protocols: `https://winprotocoldocs-bhdugrdyduf5h2e4.b02.azurefd.net/<SPEC>/[<SPEC>].pdf`
- SQL Server protocols: `https://sqlprotocoldocs-cgcjdngdb5dee9c6.b02.azurefd.net/<SPEC>/[<SPEC>].pdf`

The old `winprotocoldoc.blob.core.windows.net` archive is dead (public access was
disabled). Use the Front Door hosts above.

## Licensing

Microsoft's Intellectual Property Rights Notice for Open Specifications Documentation
(printed in each PDF) lets you make copies "in order to develop implementations of the
technologies that are described in this documentation," and distribute portions as
needed to document an implementation. The technologies are generally covered by the
Microsoft Open Specifications Promise. Keeping these PDFs here for our own TDS
implementation is within that grant. The patent grant is the Open Specifications
Promise; this is not a copyright assignment, and the PDFs remain Microsoft documents.

## Refreshing

These pin the versions we implement against. To update, pull the current PDF from the
Front Door host (or the landing page's "Published Version" row) and bump the table
above. Don't track every Microsoft revision automatically; pin deliberately, like a
dependency.
