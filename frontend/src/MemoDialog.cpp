#include "MemoDialog.h"
#include "IpcClient.h"

#include <QDialogButtonBox>
#include <QFile>
#include <QFileDialog>
#include <QHBoxLayout>
#include <QJsonArray>
#include <QLabel>
#include <QListWidget>
#include <QMessageBox>
#include <QPushButton>
#include <QTabWidget>
#include <QTextBrowser>
#include <QTextCursor>
#include <QTextDocument>
#include <QTextEdit>
#include <QVBoxLayout>
#include <functional>

namespace {
qint64 jsonInt(const QJsonValue &v) { return v.toVariant().toLongLong(); }
QString rpcMessage(const QJsonObject &error) {
    return error.value("message").toString(QObject::tr("Đã xảy ra lỗi không xác định."));
}
}

MemoDialog::MemoDialog(IpcClient *ipc, const QJsonObject &memo, QWidget *parent)
    : QDialog(parent), ipc_(ipc), memo_(memo) {
    setWindowTitle(tr("Chỉnh sửa ghi chú"));
    setModal(true);
    resize(900, 700);
    setMinimumSize(680, 520);
    buildUi();
}

void MemoDialog::buildUi() {
    auto *root = new QVBoxLayout(this);
    root->setContentsMargins(14, 14, 14, 14);
    root->setSpacing(10);

    auto *toolbar = new QHBoxLayout;
    const auto tool = [this, toolbar](const QString &text, const QString &tip, auto fn) {
        auto *button = new QPushButton(text, this);
        button->setToolTip(tip);
        button->setFixedHeight(32);
        connect(button, &QPushButton::clicked, this, fn);
        toolbar->addWidget(button);
    };
    tool("B", tr("Đậm (Markdown)"), [this] { wrapSelection("**", "**"); });
    tool("I", tr("Nghiêng (Markdown)"), [this] { wrapSelection("*", "*"); });
    tool("S", tr("Gạch ngang"), [this] { wrapSelection("~~", "~~"); });
    tool("</>", tr("Mã nguồn"), [this] { wrapSelection("`", "`"); });
    tool("•", tr("Danh sách"), [this] { prefixLines("- "); });
    tool("1.", tr("Danh sách đánh số"), [this] { prefixLines("1. "); });
    tool("H2", tr("Tiêu đề"), [this] { prefixLines("## "); });
    toolbar->addSpacing(8);
    tool(tr("Nhập .md"), tr("Nạp Markdown vào bản nháp"), [this] { importMarkdown(); });
    tool(tr("Xuất .md"), tr("Xuất bản nháp ra Markdown"), [this] { exportMarkdown(); });
    toolbar->addStretch();
    root->addLayout(toolbar);

    tabs_ = new QTabWidget(this);
    editor_ = new QTextEdit(tabs_);
    editor_->setAcceptRichText(false);
    editor_->setPlaceholderText(tr("Viết ghi chú bằng văn bản hoặc Markdown..."));
    editor_->setPlainText(memo_.value("content").toString());
    preview_ = new QTextBrowser(tabs_);
    preview_->setOpenExternalLinks(false);
    preview_->setOpenLinks(false);
    tabs_->addTab(editor_, tr("Soạn thảo"));
    tabs_->addTab(preview_, tr("Xem trước"));
    connect(tabs_, &QTabWidget::currentChanged, this, [this](int index) { if (index == 1) refreshPreview(); });
    root->addWidget(tabs_, 1);

    auto *attachmentHeader = new QHBoxLayout;
    auto *label = new QLabel(tr("Tệp đính kèm"), this);
    label->setObjectName("sectionTitle");
    attachmentHeader->addWidget(label);
    attachmentHeader->addStretch();
    auto *add = new QPushButton(tr("Thêm"), this);
    auto *up = new QPushButton(tr("Lên"), this);
    auto *down = new QPushButton(tr("Xuống"), this);
    auto *remove = new QPushButton(tr("Xóa tệp"), this);
    remove->setObjectName("danger");
    connect(add, &QPushButton::clicked, this, &MemoDialog::addAttachments);
    connect(up, &QPushButton::clicked, this, [this] { moveAttachment(-1); });
    connect(down, &QPushButton::clicked, this, [this] { moveAttachment(1); });
    connect(remove, &QPushButton::clicked, this, &MemoDialog::removeSelectedAttachment);
    attachmentHeader->addWidget(add);
    attachmentHeader->addWidget(up);
    attachmentHeader->addWidget(down);
    attachmentHeader->addWidget(remove);
    root->addLayout(attachmentHeader);

    attachments_ = new QListWidget(this);
    attachments_->setMaximumHeight(140);
    root->addWidget(attachments_);
    refreshAttachments();

    auto *buttons = new QDialogButtonBox(QDialogButtonBox::Save | QDialogButtonBox::Cancel, this);
    buttons->button(QDialogButtonBox::Save)->setText(tr("Lưu"));
    buttons->button(QDialogButtonBox::Save)->setObjectName("primary");
    buttons->button(QDialogButtonBox::Cancel)->setText(tr("Đóng"));
    connect(buttons->button(QDialogButtonBox::Save), &QPushButton::clicked, this, &MemoDialog::saveMemo);
    connect(buttons, &QDialogButtonBox::rejected, this, &QDialog::reject);
    root->addWidget(buttons);
}

void MemoDialog::refreshPreview() {
    preview_->document()->setMarkdown(editor_->toPlainText(), QTextDocument::MarkdownDialectGitHub);
}

void MemoDialog::refreshAttachments() {
    attachments_->clear();
    const QJsonArray list = memo_.value("attachments").toArray();
    for (const QJsonValue &value : list) {
        const QJsonObject a = value.toObject();
        auto *item = new QListWidgetItem(QString("%1   (%2)").arg(a.value("filename").toString(), a.value("mimeType").toString()), attachments_);
        item->setData(Qt::UserRole, a.value("id").toVariant());
        item->setToolTip(a.value("localPath").toString());
    }
}

void MemoDialog::wrapSelection(const QString &left, const QString &right) {
    QTextCursor cursor = editor_->textCursor();
    const QString selected = cursor.selectedText();
    cursor.insertText(left + selected + right);
    if (selected.isEmpty()) cursor.movePosition(QTextCursor::Left, QTextCursor::MoveAnchor, right.size());
    editor_->setTextCursor(cursor);
    editor_->setFocus();
}

void MemoDialog::prefixLines(const QString &prefix) {
    QTextCursor cursor = editor_->textCursor();
    if (!cursor.hasSelection()) {
        cursor.movePosition(QTextCursor::StartOfBlock);
        cursor.insertText(prefix);
        return;
    }
    int start = cursor.selectionStart();
    int end = cursor.selectionEnd();
    cursor.setPosition(start);
    cursor.movePosition(QTextCursor::StartOfBlock);
    while (cursor.position() <= end) {
        cursor.insertText(prefix);
        end += prefix.size();
        if (!cursor.movePosition(QTextCursor::NextBlock)) break;
    }
}

void MemoDialog::saveMemo() {
    const QJsonObject params{{"id", memoId()}, {"revision", revision()}, {"content", editor_->toPlainText()}};
    ipc_->call("memo.update", params, [this](const QJsonValue &result, const QJsonObject &error) {
        if (!error.isEmpty()) {
            if (error.value("code").toInt() == 409) {
                const auto answer = QMessageBox::question(this, tr("Ghi chú đã thay đổi"),
                    tr("Ghi chú đã được cập nhật ở nơi khác. Tải lại phiên bản mới nhất?"));
                if (answer == QMessageBox::Yes) reloadMemo();
            } else {
                QMessageBox::critical(this, tr("Không thể lưu"), rpcMessage(error));
            }
            return;
        }
        memo_ = result.toObject();
        emit saved(memo_);
        emit changed();
        accept();
    });
}

void MemoDialog::reloadMemo() {
    ipc_->call("memo.get", QJsonObject{{"id", memoId()}}, [this](const QJsonValue &result, const QJsonObject &error) {
        if (!error.isEmpty()) {
            QMessageBox::critical(this, tr("Không thể tải lại"), rpcMessage(error));
            return;
        }
        memo_ = result.toObject();
        editor_->setPlainText(memo_.value("content").toString());
        refreshAttachments();
        refreshPreview();
    });
}

void MemoDialog::addAttachments() {
    const QStringList files = QFileDialog::getOpenFileNames(this, tr("Đính kèm tệp"));
    if (files.isEmpty()) return;

    auto *index = new int(0);
    auto *worker = new std::function<void()>;
    *worker = [this, files, index, worker]() {
        if (*index >= files.size()) {
            delete index;
            delete worker;
            reloadMemo();
            emit changed();
            return;
        }
        const QString path = files.at((*index)++);
        ipc_->call("attachment.add", QJsonObject{{"memoId", memoId()}, {"path", path}},
                   [this, path, worker](const QJsonValue &, const QJsonObject &error) {
            if (!error.isEmpty()) QMessageBox::warning(this, tr("Không thể đính kèm"), tr("%1\n\n%2").arg(path, rpcMessage(error)));
            (*worker)();
        });
    };
    (*worker)();
}

void MemoDialog::removeSelectedAttachment() {
    auto *item = attachments_->currentItem();
    if (!item) return;
    const qint64 id = item->data(Qt::UserRole).toLongLong();
    if (QMessageBox::question(this, tr("Xóa tệp"), tr("Xóa tệp đính kèm này khỏi ghi chú?")) != QMessageBox::Yes) return;
    ipc_->call("attachment.delete", QJsonObject{{"id", id}}, [this](const QJsonValue &, const QJsonObject &error) {
        if (!error.isEmpty()) QMessageBox::critical(this, tr("Không thể xóa"), rpcMessage(error));
        else { reloadMemo(); emit changed(); }
    });
}

void MemoDialog::moveAttachment(int delta) {
    const int row = attachments_->currentRow();
    if (row < 0) return;
    const int target = row + delta;
    if (target < 0 || target >= attachments_->count()) return;
    QListWidgetItem *item = attachments_->takeItem(row);
    attachments_->insertItem(target, item);
    attachments_->setCurrentRow(target);
    persistAttachmentOrder();
}

void MemoDialog::persistAttachmentOrder() {
    QJsonArray ids;
    for (int i = 0; i < attachments_->count(); ++i) ids.append(static_cast<double>(attachments_->item(i)->data(Qt::UserRole).toLongLong()));
    ipc_->call("attachment.reorder", QJsonObject{{"memoId", memoId()}, {"ids", ids}}, [this](const QJsonValue &, const QJsonObject &error) {
        if (!error.isEmpty()) {
            QMessageBox::warning(this, tr("Không thể sắp xếp"), rpcMessage(error));
            reloadMemo();
        } else {
            emit changed();
        }
    });
}

void MemoDialog::importMarkdown() {
    const QString path = QFileDialog::getOpenFileName(this, tr("Nhập Markdown"), {}, tr("Markdown (*.md *.markdown);;Text (*.txt);;Tất cả tệp (*)"));
    if (path.isEmpty()) return;
    QFile file(path);
    if (!file.open(QIODevice::ReadOnly)) {
        QMessageBox::critical(this, tr("Không thể mở tệp"), file.errorString());
        return;
    }
    editor_->setPlainText(QString::fromUtf8(file.readAll()));
}

void MemoDialog::exportMarkdown() {
    const QString path = QFileDialog::getSaveFileName(this, tr("Xuất Markdown"), "note.md", tr("Markdown (*.md)"));
    if (path.isEmpty()) return;
    QFile file(path);
    if (!file.open(QIODevice::WriteOnly | QIODevice::Truncate)) {
        QMessageBox::critical(this, tr("Không thể ghi tệp"), file.errorString());
        return;
    }
    file.write(editor_->toPlainText().toUtf8());
}

qint64 MemoDialog::memoId() const { return jsonInt(memo_.value("id")); }
qint64 MemoDialog::revision() const { return jsonInt(memo_.value("revision")); }
