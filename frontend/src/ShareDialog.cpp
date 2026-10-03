#include "ShareDialog.h"
#include "IpcClient.h"

#include <QApplication>
#include <QClipboard>
#include <QComboBox>
#include <QDateTime>
#include <QFrame>
#include <QHBoxLayout>
#include <QJsonArray>
#include <QJsonObject>
#include <QJsonValue>
#include <QLabel>
#include <QLineEdit>
#include <QMessageBox>
#include <QPushButton>
#include <QScrollArea>
#include <QVBoxLayout>

namespace {
QString rpcMessage(const QJsonObject &error) {
    return error.value("message").toString(QObject::tr("Đã xảy ra lỗi không xác định."));
}

QString expiryText(const QJsonObject &share) {
    const QString raw = share.value("expiresAt").toString();
    if (raw.isEmpty()) return QObject::tr("Không hết hạn");
    QDateTime expiry = QDateTime::fromString(raw, Qt::ISODate).toLocalTime();
    if (!expiry.isValid()) return raw;
    const QString when = expiry.toString("dd/MM/yyyy HH:mm");
    if (expiry <= QDateTime::currentDateTime()) return QObject::tr("Đã hết hạn · %1").arg(when);
    return QObject::tr("Hết hạn · %1").arg(when);
}
}

ShareDialog::ShareDialog(IpcClient *ipc, qint64 memoId, QWidget *parent)
    : QDialog(parent), ipc_(ipc), memoId_(memoId) {
    setWindowTitle(tr("Chia sẻ ghi chú"));
    setModal(false);
    resize(680, 520);
    setMinimumSize(560, 420);

    auto *root = new QVBoxLayout(this);
    root->setContentsMargins(14, 14, 14, 14);
    root->setSpacing(10);

    auto *info = new QLabel(tr("Token chỉ hiển thị đầy đủ một lần khi tạo. Nếu local share server đang bật trong Cài đặt, NoteHub trả về liên kết http://127.0.0.1/...; nếu server tắt, hãy sao chép token ngay."), this);
    info->setWordWrap(true);
    info->setObjectName("muted");
    root->addWidget(info);

    auto *createRow = new QHBoxLayout;
    expiry_ = new QComboBox(this);
    expiry_->addItem(tr("Không hết hạn"), 0);
    expiry_->addItem(tr("1 ngày"), 1);
    expiry_->addItem(tr("7 ngày"), 7);
    expiry_->addItem(tr("30 ngày"), 30);
    expiry_->setCurrentIndex(2);
    create_ = new QPushButton(tr("Tạo chia sẻ"), this);
    create_->setObjectName("primary");
    createRow->addWidget(new QLabel(tr("Thời hạn"), this));
    createRow->addWidget(expiry_);
    createRow->addStretch();
    createRow->addWidget(create_);
    root->addLayout(createRow);

    auto *linkRow = new QHBoxLayout;
    newLink_ = new QLineEdit(this);
    newLink_->setReadOnly(true);
    newLink_->setPlaceholderText(tr("Token/liên kết mới sẽ xuất hiện ở đây"));
    copy_ = new QPushButton(tr("Sao chép"), this);
    copy_->setEnabled(false);
    linkRow->addWidget(newLink_, 1);
    linkRow->addWidget(copy_);
    root->addLayout(linkRow);

    status_ = new QLabel(this);
    status_->setObjectName("muted");
    root->addWidget(status_);

    auto *section = new QLabel(tr("Các quyền chia sẻ đã tạo"), this);
    section->setObjectName("sectionTitle");
    root->addWidget(section);

    auto *body = new QWidget(this);
    sharesLayout_ = new QVBoxLayout(body);
    sharesLayout_->setContentsMargins(2, 2, 2, 2);
    sharesLayout_->setSpacing(8);
    sharesLayout_->setAlignment(Qt::AlignTop);
    auto *scroll = new QScrollArea(this);
    scroll->setWidgetResizable(true);
    scroll->setFrameShape(QFrame::NoFrame);
    scroll->setWidget(body);
    root->addWidget(scroll, 1);

    auto *close = new QPushButton(tr("Đóng"), this);
    auto *bottom = new QHBoxLayout;
    bottom->addStretch();
    bottom->addWidget(close);
    root->addLayout(bottom);

    connect(create_, &QPushButton::clicked, this, &ShareDialog::createShare);
    connect(copy_, &QPushButton::clicked, this, &ShareDialog::copyCurrentLink);
    connect(close, &QPushButton::clicked, this, &QDialog::close);

    refreshShares();
}

void ShareDialog::createShare() {
    create_->setEnabled(false);
    status_->setText(tr("Đang tạo…"));
    const int days = expiry_->currentData().toInt();
    ipc_->call("share.create", QJsonObject{{"memoId", memoId_}, {"expiresDays", days}},
               [this](const QJsonValue &result, const QJsonObject &error) {
        create_->setEnabled(true);
        if (!error.isEmpty()) {
            status_->setText(tr("Không thể tạo chia sẻ"));
            QMessageBox::critical(this, tr("Không thể tạo chia sẻ"), rpcMessage(error));
            return;
        }
        const QJsonObject grant = result.toObject();
        const QString url = grant.value("url").toString();
        const QString token = grant.value("token").toString();
        newLink_->setText(url.isEmpty() ? token : url);
        newLink_->selectAll();
        copy_->setEnabled(!newLink_->text().isEmpty());
        status_->setText(url.isEmpty()
            ? tr("Server chia sẻ đang tắt; đây là token. Hãy sao chép ngay.")
            : tr("Đã tạo liên kết local. Hãy sao chép ngay."));
        emit changed();
        refreshShares();
    });
}

void ShareDialog::refreshShares() {
    ipc_->call("share.list", QJsonObject{{"memoId", memoId_}}, [this](const QJsonValue &result, const QJsonObject &error) {
        if (!error.isEmpty()) {
            status_->setText(rpcMessage(error));
            return;
        }
        rebuildShares(result.toObject().value("items").toArray());
    });
}

void ShareDialog::rebuildShares(const QJsonArray &items) {
    while (auto *item = sharesLayout_->takeAt(0)) {
        if (auto *widget = item->widget()) widget->deleteLater();
        delete item;
    }
    if (items.isEmpty()) {
        auto *empty = new QLabel(tr("Chưa có quyền chia sẻ nào."), this);
        empty->setObjectName("muted");
        sharesLayout_->addWidget(empty);
        return;
    }

    for (const QJsonValue &value : items) {
        const QJsonObject share = value.toObject();
        const QString uid = share.value("uid").toString();
        auto *row = new QFrame(this);
        row->setObjectName("card");
        auto *layout = new QHBoxLayout(row);
        layout->setContentsMargins(10, 7, 10, 7);
        const QString shortUid = uid.left(8) + (uid.size() > 8 ? QStringLiteral("…") : QString());
        auto *label = new QLabel(QString("%1   ·   %2").arg(shortUid, expiryText(share)), row);
        label->setTextInteractionFlags(Qt::TextSelectableByMouse);
        auto *revoke = new QPushButton(tr("Thu hồi"), row);
        revoke->setObjectName("danger");
        layout->addWidget(label, 1);
        layout->addWidget(revoke);
        connect(revoke, &QPushButton::clicked, this, [this, uid, revoke] {
            if (QMessageBox::question(this, tr("Thu hồi chia sẻ"), tr("Thu hồi quyền chia sẻ này? Liên kết/token cũ sẽ không còn truy cập được.")) != QMessageBox::Yes) return;
            revoke->setEnabled(false);
            ipc_->call("share.revoke", QJsonObject{{"uid", uid}}, [this](const QJsonValue &, const QJsonObject &error) {
                if (!error.isEmpty()) QMessageBox::critical(this, tr("Không thể thu hồi"), rpcMessage(error));
                else { emit changed(); refreshShares(); }
            });
        });
        sharesLayout_->addWidget(row);
    }
}

void ShareDialog::copyCurrentLink() {
    const QString text = newLink_->text();
    if (text.isEmpty()) return;
    QApplication::clipboard()->setText(text);
    status_->setText(tr("Đã sao chép vào clipboard."));
}
