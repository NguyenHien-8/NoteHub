#include "NoteEditor.h"
#include <QIconEngine>
#include <QMimeData>
#include <QPainter>
#include <QStringConverter>
#include <QTextList>
#include <QtWidgets>

namespace {
enum class EditorGlyph {
    Bold,
    Italic,
    Underline,
    Strike,
    TextColor,
    Highlight,
    ClearFormat,
    Bullets,
    Numbered,
    IndentMore,
    IndentLess,
    Undo,
    Redo,
    Attachment
};

QString normalizedLines(QString value) {
    value.replace("\r\n", "\n");
    value.replace('\r', '\n');
    value.replace(QChar::ParagraphSeparator, '\n');
    value.replace(QChar::LineSeparator, '\n');
    return value;
}

void drawEditorGlyph(QPainter &p, EditorGlyph glyph, const QColor &color, const QRectF &bounds) {
    p.save();
    p.setRenderHint(QPainter::Antialiasing);
    const qreal scale = qMin(bounds.width(), bounds.height()) / 24.0;
    p.translate(bounds.center().x() - 12.0 * scale, bounds.center().y() - 12.0 * scale);
    p.scale(scale, scale);
    QPen pen(color, 1.65, Qt::SolidLine, Qt::RoundCap, Qt::RoundJoin);
    p.setPen(pen);
    p.setBrush(Qt::NoBrush);

    auto drawLetter = [&](const QString &text, bool bold = false, bool italic = false) {
        QFont font = qApp ? qApp->font() : QFont();
        font.setPixelSize(15);
        font.setBold(bold);
        font.setItalic(italic);
        p.setFont(font);
        p.drawText(QRectF(3, 2, 18, 19), Qt::AlignCenter, text);
    };

    switch (glyph) {
    case EditorGlyph::Bold:
        drawLetter("B", true, false);
        break;
    case EditorGlyph::Italic:
        drawLetter("I", false, true);
        break;
    case EditorGlyph::Underline:
        drawLetter("U");
        p.drawLine(QPointF(6, 20), QPointF(18, 20));
        break;
    case EditorGlyph::Strike:
        drawLetter("S");
        p.drawLine(QPointF(5, 12), QPointF(19, 12));
        break;
    case EditorGlyph::TextColor:
        drawLetter("A", true, false);
        p.setPen(QPen(color, 2.4, Qt::SolidLine, Qt::RoundCap));
        p.drawLine(QPointF(6, 20), QPointF(18, 20));
        break;
    case EditorGlyph::Highlight: {
        QPainterPath marker;
        marker.moveTo(7, 5);
        marker.lineTo(18, 16);
        marker.lineTo(14.5, 19.5);
        marker.lineTo(3.5, 8.5);
        marker.closeSubpath();
        p.drawPath(marker);
        p.drawLine(QPointF(5, 20.5), QPointF(19, 20.5));
        break;
    }
    case EditorGlyph::ClearFormat:
        drawLetter("T", true, false);
        p.drawLine(QPointF(15.5, 15.5), QPointF(20, 20));
        p.drawLine(QPointF(20, 15.5), QPointF(15.5, 20));
        break;
    case EditorGlyph::Bullets:
        p.setBrush(color);
        p.setPen(Qt::NoPen);
        for (qreal y : {6.5, 12.0, 17.5})
            p.drawEllipse(QRectF(4, y - 1.2, 2.4, 2.4));
        p.setPen(pen);
        for (qreal y : {6.5, 12.0, 17.5})
            p.drawLine(QPointF(9, y), QPointF(20, y));
        break;
    case EditorGlyph::Numbered: {
        QFont font = qApp ? qApp->font() : QFont();
        font.setPixelSize(7);
        p.setFont(font);
        for (int i = 0; i < 3; ++i) {
            const qreal y = 6.0 + i * 5.7;
            p.drawText(QRectF(2.5, y - 4, 5, 7), Qt::AlignCenter, QString::number(i + 1));
            p.drawLine(QPointF(9, y), QPointF(20, y));
        }
        break;
    }
    case EditorGlyph::IndentMore:
        p.drawLine(QPointF(10, 6), QPointF(20, 6));
        p.drawLine(QPointF(10, 12), QPointF(20, 12));
        p.drawLine(QPointF(10, 18), QPointF(20, 18));
        p.drawPolyline(QPolygonF{QPointF(4, 8), QPointF(8, 12), QPointF(4, 16)});
        break;
    case EditorGlyph::IndentLess:
        p.drawLine(QPointF(10, 6), QPointF(20, 6));
        p.drawLine(QPointF(10, 12), QPointF(20, 12));
        p.drawLine(QPointF(10, 18), QPointF(20, 18));
        p.drawPolyline(QPolygonF{QPointF(8, 8), QPointF(4, 12), QPointF(8, 16)});
        break;
    case EditorGlyph::Undo: {
        QPainterPath path;
        path.moveTo(8, 8);
        path.cubicTo(12, 4, 20, 6, 20, 13);
        path.cubicTo(20, 18, 15.5, 20, 11, 19);
        p.drawPath(path);
        p.drawPolyline(QPolygonF{QPointF(8, 4), QPointF(8, 9), QPointF(3, 9)});
        break;
    }
    case EditorGlyph::Redo: {
        QPainterPath path;
        path.moveTo(16, 8);
        path.cubicTo(12, 4, 4, 6, 4, 13);
        path.cubicTo(4, 18, 8.5, 20, 13, 19);
        p.drawPath(path);
        p.drawPolyline(QPolygonF{QPointF(16, 4), QPointF(16, 9), QPointF(21, 9)});
        break;
    }
    case EditorGlyph::Attachment: {
        QPainterPath clip;
        clip.moveTo(8.3, 12.7);
        clip.lineTo(14.8, 6.2);
        clip.cubicTo(17.2, 3.8, 20.9, 7.5, 18.5, 9.9);
        clip.lineTo(10.8, 17.6);
        clip.cubicTo(7.0, 21.4, 1.8, 16.2, 5.5, 12.5);
        clip.lineTo(13.4, 4.6);
        p.drawPath(clip);
        break;
    }
    }
    p.restore();
}

class EditorIconEngine final : public QIconEngine {
  public:
    explicit EditorIconEngine(EditorGlyph value) : glyph(value) {}
    QIconEngine *clone() const override {
        return new EditorIconEngine(glyph);
    }
    void paint(QPainter *painter, const QRect &rect, QIcon::Mode mode, QIcon::State) override {
        const auto palette = qApp ? qApp->palette() : QPalette();
        const auto group = mode == QIcon::Disabled ? QPalette::Disabled : QPalette::Active;
        const QColor color = palette.color(group, QPalette::Text);
        drawEditorGlyph(*painter, glyph, color, rect.adjusted(2, 2, -2, -2));
    }
    QPixmap pixmap(const QSize &size, QIcon::Mode mode, QIcon::State state) override {
        QPixmap pixmap(size);
        pixmap.fill(Qt::transparent);
        QPainter painter(&pixmap);
        paint(&painter, pixmap.rect(), mode, state);
        return pixmap;
    }

  private:
    EditorGlyph glyph;
};

QIcon editorIcon(EditorGlyph glyph) {
    return QIcon(new EditorIconEngine(glyph));
}

class RichPasteTextEdit final : public QTextEdit {
  public:
    using QTextEdit::QTextEdit;

  protected:
    void insertFromMimeData(const QMimeData *source) override {
        if (!source)
            return;

        // Prefer rich clipboard HTML when Qt can reproduce the same line
        // structure. Word/ChatGPT HTML therefore keeps bold, lists, colors,
        // etc. If an application's HTML collapses visual newlines (seen with
        // some chat clients), fall back to the clipboard's plain text so code
        // and command blocks never turn into one long line after saving.
        const QString clipboardText = normalizedLines(source->text());
        if (source->hasHtml()) {
            QTextDocument probe;
            probe.setHtml(source->html());
            const QString htmlText = normalizedLines(probe.toPlainText());
            const bool hasMultipleLines = clipboardText.contains('\n');
            const bool htmlKeepsLines = !hasMultipleLines || htmlText.count('\n') >= clipboardText.count('\n');
            if (htmlKeepsLines) {
                auto cursor = textCursor();
                cursor.insertHtml(source->html());
                setTextCursor(cursor);
                ensureCursorVisible();
                return;
            }
        }
        if (source->hasText()) {
            insertPlainText(clipboardText);
            ensureCursorVisible();
            return;
        }
        QTextEdit::insertFromMimeData(source);
    }
};

QString stableHtml(const QTextDocument *document) {
    const QString html = document->toHtml();
    QTextDocument roundTrip;
    roundTrip.setHtml(html);
    const QString before = normalizedLines(document->toPlainText());
    const QString after = normalizedLines(roundTrip.toPlainText());
    if (before.count('\n') == after.count('\n'))
        return html;

    // This path is deliberately conservative. It is only used if Qt's own HTML
    // serializer loses visible line boundaries. Ordinary whitespace normalization
    // is tolerated so rich formatting is retained whenever line structure is safe.
    QTextDocument safe;
    safe.setPlainText(before);
    return safe.toHtml();
}
} // namespace

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
    toolbar->setToolButtonStyle(Qt::ToolButtonIconOnly);
    toolbar->setIconSize({19, 19});
    layout->addWidget(toolbar);

    rich = new RichPasteTextEdit(this);
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

    auto action = [this](EditorGlyph glyph, const QString &label, const QString &hint,
                         const QKeySequence &shortcut, auto callback) {
        auto a = toolbar->addAction(editorIcon(glyph), label);
        a->setToolTip(hint);
        a->setShortcut(shortcut);
        a->setShortcutContext(Qt::WidgetWithChildrenShortcut);
        addAction(a);
        connect(a, &QAction::triggered, this, callback);
        return a;
    };

    action(EditorGlyph::Bold, tr("Bold"), tr("Bold · Ctrl+B"), QKeySequence::Bold, [this] {
        QTextCharFormat f;
        f.setFontWeight(rich->fontWeight() == QFont::Bold ? QFont::Normal : QFont::Bold);
        format(f);
    });
    action(EditorGlyph::Italic, tr("Italic"), tr("Italic · Ctrl+I"), QKeySequence::Italic, [this] {
        QTextCharFormat f;
        f.setFontItalic(!rich->fontItalic());
        format(f);
    });
    action(EditorGlyph::Underline, tr("Underline"), tr("Underline · Ctrl+U"), QKeySequence::Underline, [this] {
        QTextCharFormat f;
        f.setFontUnderline(!rich->fontUnderline());
        format(f);
    });
    action(EditorGlyph::Strike, tr("Strikethrough"), tr("Strikethrough"), {}, [this] {
        QTextCharFormat f;
        f.setFontStrikeOut(!rich->currentCharFormat().fontStrikeOut());
        format(f);
    });

    toolbar->addSeparator();
    auto size = new QSpinBox(toolbar);
    size->setObjectName("editorFontSize");
    size->setRange(8, 72);
    size->setValue(12);
    size->setSuffix(" pt");
    size->setToolTip(tr("Font size"));
    size->setFocusPolicy(Qt::ClickFocus);
    size->setFixedWidth(92);
    toolbar->addWidget(size);
    connect(size, &QSpinBox::valueChanged, this, [this](int n) {
        QTextCharFormat f;
        f.setFontPointSize(n);
        format(f);
    });

    action(EditorGlyph::TextColor, tr("Text color"), tr("Text color"), {}, [this] {
        auto c = QColorDialog::getColor(rich->textColor(), this, tr("Text color"));
        if (c.isValid()) {
            QTextCharFormat f;
            f.setForeground(c);
            format(f);
        }
    });
    action(EditorGlyph::Highlight, tr("Highlight"), tr("Highlight"), {}, [this] {
        auto c = QColorDialog::getColor(QColor("#fff2a8"), this, tr("Highlight"));
        if (c.isValid()) {
            QTextCharFormat f;
            f.setBackground(c);
            format(f);
        }
    });
    action(EditorGlyph::ClearFormat, tr("Clear formatting"), tr("Clear formatting (selection, or whole note)"), {},
           [this] {
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
    action(EditorGlyph::Bullets, tr("Bulleted list"), tr("Bulleted list"), {},
           [this] { list(QTextListFormat::ListDisc); });
    action(EditorGlyph::Numbered, tr("Numbered list"), tr("Numbered list"), {},
           [this] { list(QTextListFormat::ListDecimal); });
    action(EditorGlyph::IndentMore, tr("Increase indent"), tr("Increase indent"), {}, [this] { indent(1); });
    action(EditorGlyph::IndentLess, tr("Decrease indent"), tr("Decrease indent"), {}, [this] { indent(-1); });

    toolbar->addSeparator();
    action(EditorGlyph::Undo, tr("Undo"), tr("Undo · Ctrl+Z"), QKeySequence::Undo, [this] { rich->undo(); });
    action(EditorGlyph::Redo, tr("Redo"), tr("Redo · Ctrl+Y"), QKeySequence::Redo, [this] { rich->redo(); });

    toolbar->addSeparator();
    action(EditorGlyph::Attachment, tr("Attach files"), tr("Attach files to this note"), {},
           [this] { emit attachRequested(); });

    auto files = new QHBoxLayout;
    auto import = new QPushButton(tr("Import .md"), this), exportFile = new QPushButton(tr("Export .md"), this);
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
    return RichTextPrefix + stableHtml(rich->document());
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
    if (modified() && QMessageBox::question(this, tr("Replace draft?"),
                                             tr("Replace the current draft with this Markdown file?")) !=
                          QMessageBox::Yes)
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
