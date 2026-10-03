#include "NoteEditor.h"
#include <QStringConverter>
#include <QTextList>
#include <QtWidgets>

void loadNoteDocument(QTextDocument *document, const QString &content) {
    if (content.startsWith(RichTextPrefix))
        document->setHtml(content.mid(RichTextPrefix.size()));
    else
        document->setMarkdown(content, QTextDocument::MarkdownDialectGitHub);
    document->setModified(false);
}

NoteEditor::NoteEditor(QWidget *parent) : QWidget(parent) {
    auto layout = new QVBoxLayout(this);
    layout->setContentsMargins(0, 0, 0, 0);
    layout->setSpacing(8);
    toolbar = new QToolBar(this);
    toolbar->setObjectName("editorToolbar");
    toolbar->setMovable(false);
    toolbar->setFloatable(false);
    toolbar->setIconSize({18, 18});
    layout->addWidget(toolbar);
    rich = new QTextEdit(this);
    rich->setObjectName("editorSurface");
    rich->setDocument(new LocalTextDocument(rich));
    rich->setAcceptRichText(true);
    rich->setPlaceholderText(tr("Write your note…"));
    rich->setMinimumHeight(140);
    source = new QPlainTextEdit(this);
    source->setObjectName("editorSource");
    source->setPlaceholderText(tr("# Markdown source"));
    source->setFont(QFontDatabase::systemFont(QFontDatabase::FixedFont));
    tabs = new QTabWidget(this);
    tabs->setObjectName("editorTabs");
    tabs->setDocumentMode(true);
    tabs->addTab(rich, tr("Write"));
    tabs->addTab(source, tr("Markdown source"));
    layout->addWidget(tabs, 1);
    auto markdownHint = new QLabel(tr("Editing Markdown source converts this note to Markdown. "
                                      "Colors, highlights, underline and font sizes may be lost."), this);
    markdownHint->setWordWrap(true);
    markdownHint->setObjectName("muted");
    markdownHint->setVisible(false);
    layout->addWidget(markdownHint);
    connect(tabs, &QTabWidget::currentChanged, markdownHint,
            [markdownHint](int index) { markdownHint->setVisible(index == 1); });
    auto action = [this](const QString &label, const QString &hint, const QKeySequence &shortcut,
                         auto callback) {
        auto a = toolbar->addAction(label);
        a->setToolTip(hint);
        a->setShortcut(shortcut);
        a->setShortcutContext(Qt::WidgetWithChildrenShortcut);
        addAction(a);
        connect(a, &QAction::triggered, this, callback);
        return a;
    };
    action("B", tr("Bold · Ctrl+B"), QKeySequence::Bold, [this] {
        QTextCharFormat f;
        f.setFontWeight(rich->fontWeight() == QFont::Bold ? QFont::Normal : QFont::Bold);
        format(f);
    });
    action("I", tr("Italic · Ctrl+I"), QKeySequence::Italic, [this] {
        QTextCharFormat f;
        f.setFontItalic(!rich->fontItalic());
        format(f);
    });
    action("U", tr("Underline · Ctrl+U"), QKeySequence::Underline, [this] {
        QTextCharFormat f;
        f.setFontUnderline(!rich->fontUnderline());
        format(f);
    });
    action("S̶", tr("Strikethrough"), {}, [this] {
        QTextCharFormat f;
        f.setFontStrikeOut(!rich->currentCharFormat().fontStrikeOut());
        format(f);
    });
    toolbar->addSeparator();
    auto size = new QSpinBox(toolbar);
    size->setRange(8, 72);
    size->setValue(12);
    size->setSuffix(" pt");
    size->setToolTip(tr("Font size"));
    size->setFocusPolicy(Qt::ClickFocus);
    toolbar->addWidget(size);
    connect(size, &QSpinBox::valueChanged, this, [this](int n) {
        QTextCharFormat f;
        f.setFontPointSize(n);
        format(f);
    });
    action("A", tr("Text color"), {}, [this] {
        auto c = QColorDialog::getColor(rich->textColor(), this, tr("Text color"));
        if (c.isValid()) {
            QTextCharFormat f;
            f.setForeground(c);
            format(f);
        }
    });
    action("▰", tr("Highlight"), {}, [this] {
        auto c = QColorDialog::getColor(QColor("#fff2a8"), this, tr("Highlight"));
        if (c.isValid()) {
            QTextCharFormat f;
            f.setBackground(c);
            format(f);
        }
    });
    action("Tx", tr("Clear formatting (selection, or whole note)"), {}, [this] {
        auto c = rich->textCursor();
        if (!c.hasSelection())
            c.select(QTextCursor::Document);
        c.beginEditBlock();
        QTextCharFormat f;
        f.setFont(rich->document()->defaultFont());
        c.setCharFormat(f);
        const int end = c.selectionEnd();
        c.setPosition(c.selectionStart());
        do {
            if (auto l = c.currentList())
                l->remove(c.block());
            c.setBlockFormat(QTextBlockFormat());
        } while (c.movePosition(QTextCursor::NextBlock) && c.position() < end);
        c.endEditBlock();
        rich->setFocus();
    });
    toolbar->addSeparator();
    action("•", tr("Bulleted list"), {}, [this] { list(QTextListFormat::ListDisc); });
    action("1.", tr("Numbered list"), {}, [this] { list(QTextListFormat::ListDecimal); });
    action("→|", tr("Increase indent"), {}, [this] { indent(1); });
    action("|←", tr("Decrease indent"), {}, [this] { indent(-1); });
    toolbar->addSeparator();
    action("↶", tr("Undo"), {}, [this] { rich->undo(); });
    action("↷", tr("Redo"), {}, [this] { rich->redo(); });
    auto files = new QHBoxLayout;
    auto import = new QPushButton(tr("Import .md"), this),
         exportFile = new QPushButton(tr("Export .md"), this);
    import->setObjectName("outlineButton");
    exportFile->setObjectName("outlineButton");
    files->addWidget(import);
    files->addWidget(exportFile);
    files->addStretch();
    layout->addLayout(files);
    connect(import, &QPushButton::clicked, this, &NoteEditor::importMarkdown);
    connect(exportFile, &QPushButton::clicked, this, &NoteEditor::exportMarkdown);
    connect(tabs, &QTabWidget::currentChanged, this, [this](int index) {
        if (loading)
            return;
        toolbar->setEnabled(index == 0);
        if (index == 1) {
            QSignalBlocker blocker(source);
            source->setPlainText(rich->document()->toMarkdown());
            source->document()->setModified(false);
        } else if (source->document()->isModified()) {
            rich->setMarkdown(source->toPlainText());
            rich->document()->setModified(true);
            source->document()->setModified(false);
        }
    });
    setContent({});
}
void NoteEditor::setContent(const QString &value) {
    loading = true;
    original = value;
    loadNoteDocument(rich->document(), value);
    source->clear();
    source->document()->setModified(false);
    tabs->setCurrentIndex(0);
    toolbar->setEnabled(true);
    loading = false;
}
bool NoteEditor::modified() const {
    return rich->document()->isModified() || source->document()->isModified();
}
QString NoteEditor::content() const {
    if (!modified())
        return original;
    if (tabs->currentIndex() == 1 && source->document()->isModified())
        return source->toPlainText();
    return RichTextPrefix + rich->document()->toHtml();
}
void NoteEditor::format(const QTextCharFormat &f) {
    if (tabs->currentIndex() != 0)
        return;
    auto c = rich->textCursor();
    c.mergeCharFormat(f);
    rich->mergeCurrentCharFormat(f);
    rich->setFocus();
}
void NoteEditor::list(QTextListFormat::Style style) {
    auto c = rich->textCursor();
    c.beginEditBlock();
    QTextListFormat f;
    f.setStyle(style);
    f.setIndent(c.currentList() ? c.currentList()->format().indent() : 1);
    c.createList(f);
    c.endEditBlock();
    rich->setFocus();
}
void NoteEditor::indent(int delta) {
    auto c = rich->textCursor();
    c.beginEditBlock();
    const int end = c.selectionEnd();
    c.setPosition(c.selectionStart());
    do {
        if (auto l = c.currentList()) {
            auto f = l->format();
            const int n = f.indent() + delta;
            if (n <= 0) {
                l->remove(c.block());
                auto b = c.blockFormat();
                b.setIndent(0);
                c.setBlockFormat(b);
            } else {
                f.setIndent(n);
                c.createList(f);
            }
        } else {
            auto f = c.blockFormat();
            f.setIndent(qMax(0, f.indent() + delta));
            c.setBlockFormat(f);
        }
    } while (c.movePosition(QTextCursor::NextBlock) && c.position() < end);
    c.endEditBlock();
    rich->setFocus();
}
void NoteEditor::importMarkdown() {
    auto path = QFileDialog::getOpenFileName(this, tr("Import Markdown"), {},
                                             tr("Markdown (*.md *.markdown);;Text (*.txt)"));
    if (path.isEmpty())
        return;
    QFile file(path);
    if (!file.open(QIODevice::ReadOnly) || file.size() > 2 * 1024 * 1024) {
        QMessageBox::warning(this, tr("Import failed"), tr("Choose a readable UTF-8 file up to 2 MiB."));
        return;
    }
    QStringDecoder decoder(QStringDecoder::Utf8);
    const QString text = decoder.decode(file.readAll());
    if (decoder.hasError() || text.contains(QChar(0))) {
        QMessageBox::warning(this, tr("Import failed"), tr("The file is not valid UTF-8 text."));
        return;
    }
    if (modified() &&
        QMessageBox::question(this, tr("Replace draft?"),
                              tr("Replace the current draft with this Markdown file?")) != QMessageBox::Yes)
        return;
    loading = true;
    tabs->setCurrentIndex(0);
    rich->setMarkdown(text);
    rich->document()->setModified(true);
    source->document()->setModified(false);
    toolbar->setEnabled(true);
    loading = false;
}
void NoteEditor::exportMarkdown() {
    if (QMessageBox::question(this, tr("Export Markdown"),
                              tr("Markdown preserves text, headings and lists. Custom colors, highlights and "
                                 "font sizes may be omitted. Attachments are separate files. Continue?")) !=
        QMessageBox::Yes)
        return;
    auto path = QFileDialog::getSaveFileName(this, tr("Export Markdown"), "note.md", tr("Markdown (*.md)"));
    if (path.isEmpty())
        return;
    QSaveFile file(path);
    const auto data =
        (tabs->currentIndex() == 1 && source->document()->isModified() ? source->toPlainText()
                                                                       : rich->document()->toMarkdown())
            .toUtf8();
    if (!file.open(QIODevice::WriteOnly) || file.write(data) != data.size() || !file.commit())
        QMessageBox::warning(this, tr("Export failed"), file.errorString());
}
