#pragma once

#include <QDialog>

class IpcClient;
class QJsonArray;
class QComboBox;
class QLabel;
class QLineEdit;
class QPushButton;
class QVBoxLayout;

class ShareDialog final : public QDialog {
    Q_OBJECT
public:
    ShareDialog(IpcClient *ipc, qint64 memoId, QWidget *parent = nullptr);

signals:
    void changed();

private:
    void createShare();
    void refreshShares();
    void rebuildShares(const QJsonArray &items);
    void copyCurrentLink();

    IpcClient *ipc_ = nullptr;
    qint64 memoId_ = 0;
    QComboBox *expiry_ = nullptr;
    QPushButton *create_ = nullptr;
    QLineEdit *newLink_ = nullptr;
    QPushButton *copy_ = nullptr;
    QLabel *status_ = nullptr;
    QVBoxLayout *sharesLayout_ = nullptr;
};
