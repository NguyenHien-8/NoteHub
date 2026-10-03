#include "MemoCard.h"

#include <QDateTime>
#include <QFileInfo>
#include <QGridLayout>
#include <QHBoxLayout>
#include <QIcon>
#include <QJsonArray>
#include <QLabel>
#include <QPixmap>
#include <QPushButton>
#include <QSizeF>
#include <QSizePolicy>
#include <QTextBrowser>
#include <QTextDocument>
#include <QVBoxLayout>

namespace {
qint64 asInt64(const QJsonValue &value) { return value.toVariant().toLongLong(); }

QString friendlyTime(const QString &iso) {
    QDateTime t = QDateTime::fromString(iso, Qt::ISODate);
    if (!t.isValid()) return iso;
    t = t.toLocalTime();
    return t.toString("dd/MM/yyyy  HH:mm");
}

QString humanBytes(qint64 bytes) {
    if (bytes < 1024) return QString::number(bytes) + " B";
    const double kb = bytes / 1024.0;
    if (kb < 1024) return QString::number(kb, 'f', 1) + " KB";
    return QString::number(kb / 1024.0, 'f', 1) + " MB";
}
}

MemoCard::MemoCard(const QJsonObject &memo, QWidget *parent) : QFrame(parent), memo_(memo) {
    setObjectName("card");
    setSizePolicy(QSizePolicy::Expanding, QSizePolicy::Maximum);

    auto *outer = new QVBoxLayout(this);
    outer->setContentsMargins(14, 12, 14, 12);
    outer->setSpacing(8);

    auto *header = new QHBoxLayout;
    auto *timestamp = new QLabel(friendlyTime(memo_.value("createdAt").toString()), this);
    timestamp->setObjectName("timestamp");
    header->addWidget(timestamp);
    header->addStretch();

    const bool favorite = memo_.value("favorite").toBool();
    auto *favoriteButton = new QPushButton(favorite ? QStringLiteral("★") : QStringLiteral("☆"), this);
    favoriteButton->setToolTip(favorite ? tr("Bỏ yêu thích") : tr("Yêu thích"));
    favoriteButton->setFixedWidth(38);
    connect(favoriteButton, &QPushButton::clicked, this, [this, favorite] {
        emit favoriteRequested(memoId(), !favorite);
    });
    auto *edit = new QPushButton(tr("Sửa"), this);
    auto *share = new QPushButton(tr("Chia sẻ"), this);
    auto *remove = new QPushButton(tr("Xóa"), this);
    remove->setObjectName("danger");
    connect(edit, &QPushButton::clicked, this, [this] { emit editRequested(memoId()); });
    connect(share, &QPushButton::clicked, this, [this] { emit shareRequested(memoId()); });
    connect(remove, &QPushButton::clicked, this, [this] { emit deleteRequested(memoId()); });
    header->addWidget(favoriteButton);
    header->addWidget(edit);
    header->addWidget(share);
    header->addWidget(remove);
    outer->addLayout(header);

    const QJsonArray tags = memo_.value("tags").toArray();
    if (!tags.isEmpty()) {
        QStringList values;
        for (const QJsonValue &tag : tags) values << "#" + tag.toString();
        auto *tagLabel = new QLabel(values.join("   "), this);
        tagLabel->setObjectName("muted");
        tagLabel->setTextInteractionFlags(Qt::TextSelectableByMouse);
        outer->addWidget(tagLabel);
    }

    auto *body = new QTextBrowser(this);
    body->setOpenExternalLinks(false);
    body->setOpenLinks(false);
    body->setFrameShape(QFrame::NoFrame);
    body->document()->setDocumentMargin(0);
    body->document()->setMarkdown(memo_.value("content").toString(), QTextDocument::MarkdownDialectGitHub);
    body->setHorizontalScrollBarPolicy(Qt::ScrollBarAlwaysOff);
    body->setVerticalScrollBarPolicy(Qt::ScrollBarAlwaysOff);
    body->setMaximumHeight(220);
    const QSizeF docSize = body->document()->size();
    body->setMinimumHeight(qBound(44, static_cast<int>(docSize.height()) + 8, 220));
    outer->addWidget(body);

    const QJsonArray attachments = memo_.value("attachments").toArray();
    if (!attachments.isEmpty()) {
        auto *grid = new QGridLayout;
        grid->setContentsMargins(0, 2, 0, 0);
        grid->setHorizontalSpacing(8);
        grid->setVerticalSpacing(8);
        int visible = 0;
        for (const QJsonValue &value : attachments) {
            if (!value.isObject() || visible >= 6) continue;
            const QJsonObject a = value.toObject();
            const qint64 attachmentId = asInt64(a.value("id"));
            const QString mime = a.value("mimeType").toString();
            const QString localPath = a.value("localPath").toString();
            QWidget *tile = nullptr;
            if (mime.startsWith("image/")) {
                auto *button = new QPushButton(this);
                button->setToolTip(a.value("filename").toString());
                button->setFixedSize(128, 94);
                QPixmap pix(localPath);
                if (!pix.isNull()) {
                    button->setIcon(QIcon(pix.scaled(116, 82, Qt::KeepAspectRatio, Qt::SmoothTransformation)));
                    button->setIconSize(QSize(116, 82));
                } else {
                    button->setText(tr("Ảnh"));
                }
                connect(button, &QPushButton::clicked, this, [this, attachmentId] { emit attachmentOpenRequested(attachmentId); });
                tile = button;
            } else {
                auto *button = new QPushButton(QString("%1\n%2").arg(a.value("filename").toString(), humanBytes(asInt64(a.value("size")))), this);
                button->setToolTip(localPath);
                button->setMinimumSize(150, 58);
                connect(button, &QPushButton::clicked, this, [this, attachmentId] { emit attachmentOpenRequested(attachmentId); });
                tile = button;
            }
            grid->addWidget(tile, visible / 3, visible % 3);
            ++visible;
        }
        if (attachments.size() > visible) {
            auto *more = new QLabel(tr("+ %1 tệp khác").arg(attachments.size() - visible), this);
            more->setObjectName("muted");
            grid->addWidget(more, (visible + 2) / 3, 0, 1, 3);
        }
        outer->addLayout(grid);
    }
}

qint64 MemoCard::memoId() const { return asInt64(memo_.value("id")); }
