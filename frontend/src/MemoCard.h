#pragma once

#include <QFrame>
#include <QJsonObject>

class MemoCard final : public QFrame {
    Q_OBJECT
public:
    explicit MemoCard(const QJsonObject &memo, QWidget *parent = nullptr);
    qint64 memoId() const;
    QJsonObject memo() const { return memo_; }

signals:
    void editRequested(qint64 id);
    void deleteRequested(qint64 id);
    void favoriteRequested(qint64 id, bool favorite);
    void shareRequested(qint64 id);
    void attachmentOpenRequested(qint64 attachmentId);

private:
    QJsonObject memo_;
};
