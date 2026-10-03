#pragma once
#include <QPlainTextEdit>
#include <QTabWidget>
#include <QTextDocument>
#include <QTextEdit>
#include <QToolBar>
#include <QWidget>

inline const QString RichTextPrefix = QStringLiteral("<!-- NoteHub rich text v1 -->\n");

// Notes are local documents. Inline resources are attachments managed by Go,
// so pasted/imported HTML never reads arbitrary files or network URLs.
class LocalTextDocument final : public QTextDocument {
  public:
    explicit LocalTextDocument(QObject *parent = nullptr) : QTextDocument(parent) {}

  protected:
    QVariant loadResource(int, const QUrl &) override {
        return {};
    }
};

void loadNoteDocument(QTextDocument *document, const QString &content);

class NoteEditor final : public QWidget {
    Q_OBJECT
  public:
    explicit NoteEditor(QWidget *parent = nullptr);
    void setContent(const QString &content);
    QString content() const;
    bool modified() const;
    QTextEdit *textEdit() const {
        return rich;
    }
    void importMarkdown();
    void exportMarkdown();

  private:
    QTextEdit *rich;
    QPlainTextEdit *source;
    QTabWidget *tabs;
    QToolBar *toolbar;
    QString original;
    bool loading = false;
    void format(const QTextCharFormat &format);
    void list(QTextListFormat::Style style);
    void indent(int delta);
};
