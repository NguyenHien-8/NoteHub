#pragma once

#include <QDialog>
#include <QJsonObject>

class IpcClient;
class QListWidget;
class QTabWidget;
class QTextBrowser;
class QTextEdit;

class MemoDialog final : public QDialog {
    Q_OBJECT
public:
    MemoDialog(IpcClient *ipc, const QJsonObject &memo, QWidget *parent = nullptr);

signals:
    void saved(const QJsonObject &memo);
    void changed();

private:
    void buildUi();
    void refreshPreview();
    void refreshAttachments();
    void wrapSelection(const QString &left, const QString &right);
    void prefixLines(const QString &prefix);
    void saveMemo();
    void reloadMemo();
    void addAttachments();
    void removeSelectedAttachment();
    void moveAttachment(int delta);
    void persistAttachmentOrder();
    void importMarkdown();
    void exportMarkdown();
    qint64 memoId() const;
    qint64 revision() const;

    IpcClient *ipc_ = nullptr;
    QJsonObject memo_;
    QTextEdit *editor_ = nullptr;
    QTextBrowser *preview_ = nullptr;
    QTabWidget *tabs_ = nullptr;
    QListWidget *attachments_ = nullptr;
};
