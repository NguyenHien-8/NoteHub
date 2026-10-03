#include "MainWindow.h"
#include "ImageGallery.h"
#include "NoteEditor.h"
#include <QTextList>
#include <QtWidgets>
#include <memory>

namespace {
QPushButton *button(const QString &text, QWidget *parent = nullptr) {
    auto b = new QPushButton(text, parent);
    b->setCursor(Qt::PointingHandCursor);
    return b;
}
QFrame *card() {
    auto f = new QFrame;
    f->setObjectName("card");
    f->setSizePolicy(QSizePolicy::Expanding, QSizePolicy::Preferred);
    return f;
}
QJsonArray jsonPaths(const QStringList &paths) {
    QJsonArray a;
    for (const auto &p : paths)
        a.append(p);
    return a;
}
QStringList fileFailures(const QJsonValue &result) {
    QStringList list;
    for (const auto &v : result.toObject()["failures"].toArray())
        list.append(v.toString());
    return list;
}
class MemoDialog final : public QDialog {
  public:
    using QDialog::QDialog;
    NoteEditor *editor = nullptr;
    bool busy = false;
    void reject() override {
        if (busy)
            return;
        if (editor && editor->modified() &&
            QMessageBox::question(
                this, tr("Discard edits?"),
                tr("Unsaved text will be lost. Confirmed attachment changes stay saved.")) !=
                QMessageBox::Yes)
            return;
        QDialog::reject();
    }
};
} // namespace

MainWindow::MainWindow(const QString &program, const QString &profile, QWidget *parent)
    : QMainWindow(parent), backend(this) {
    setWindowTitle("NoteHub");
    resize(1280, 820);
    setMinimumSize(720, 480);
    auto root = new QWidget(this);
    setCentralWidget(root);
    auto outer = new QVBoxLayout(root);
    outer->setContentsMargins(16, 12, 16, 12);
    outer->setSpacing(14);
    auto header = new QHBoxLayout;
    leftToggle = button("◧");
    leftToggle->setFixedWidth(38);
    leftToggle->setToolTip(tr("Collapse navigation"));
    header->addWidget(leftToggle);
    auto brand = new QLabel("NoteHub");
    brand->setObjectName("brand");
    header->addWidget(brand);
    header->addStretch();
    search = new QLineEdit;
    search->setPlaceholderText(tr("Search notes, tags and content…  Ctrl+K"));
    search->setClearButtonEnabled(true);
    search->setMaximumWidth(520);
    search->setMinimumWidth(180);
    header->addWidget(search, 1);
    outer->addLayout(header);
    auto columns = new QHBoxLayout;
    columns->setSpacing(14);
    outer->addLayout(columns, 1);
    navigation = new QFrame;
    navigation->setObjectName("navigation");
    auto nav = new QVBoxLayout(navigation);
    nav->setSpacing(8);
    nav->setContentsMargins(8, 12, 8, 12);
    const QList<QPair<QString, QStyle::StandardPixmap>> entries = {
        {"Home", QStyle::SP_DirHomeIcon},
        {"Calendar", QStyle::SP_FileDialogDetailedView},
        {"Search", QStyle::SP_FileDialogContentsView},
        {"Attachments", QStyle::SP_FileIcon},
        {"Tags", QStyle::SP_DirIcon},
        {"Settings", QStyle::SP_FileDialogInfoView}};
    for (const auto &entry : entries) {
        auto b = new QToolButton;
        b->setText(entry.first);
        b->setProperty("label", entry.first);
        b->setToolTip(entry.first);
        b->setIcon(style()->standardIcon(entry.second));
        b->setIconSize({22, 22});
        b->setCheckable(true);
        b->setMinimumHeight(42);
        b->setSizePolicy(QSizePolicy::Expanding, QSizePolicy::Fixed);
        nav->addWidget(b);
        navButtons.append(b);
        connect(b, &QToolButton::clicked, this, [this, name = entry.first.toLower()] { navigate(name); });
    }
    auto separator = new QFrame;
    separator->setFrameShape(QFrame::HLine);
    nav->addWidget(separator);
    auto myTags = new QToolButton;
    myTags->setProperty("label", "My Tags");
    myTags->setText("My Tags");
    myTags->setToolTip("My Tags");
    myTags->setIcon(style()->standardIcon(QStyle::SP_DirLinkIcon));
    myTags->setMinimumHeight(42);
    myTags->setSizePolicy(QSizePolicy::Expanding, QSizePolicy::Fixed);
    nav->addWidget(myTags);
    navButtons.append(myTags);
    nav->addStretch();
    connect(myTags, &QToolButton::clicked, this, [this, myTags] {
        QMenu menu;
        for (const auto &v : tags) {
            auto t = v.toObject();
            auto a =
                menu.addAction("#" + t["tag"].toString() + "  (" + QString::number(t["count"].toInt()) + ")");
            connect(a, &QAction::triggered, this, [this, t] {
                tag = t["tag"].toString();
                page = "home";
                refresh();
            });
        }
        menu.addSeparator();
        auto add = menu.addAction(tr("Add tag to draft…"));
        connect(add, &QAction::triggered, this, [this] {
            bool ok;
            auto value = QInputDialog::getText(this, tr("Add tag"), tr("Tag (for example Research/FPGA)"),
                                               QLineEdit::Normal, {}, &ok);
            if (ok && !value.trimmed().isEmpty()) {
                navigate("home");
                composer->insertPlainText(" #" + value.trimmed().remove('#'));
            }
        });
        menu.exec(myTags->mapToGlobal(QPoint(myTags->width(), 0)));
    });
    auto navScroll = new QScrollArea;
    navigationScroll = navScroll;
    navScroll->setWidgetResizable(true);
    navScroll->setFrameShape(QFrame::NoFrame);
    navScroll->setHorizontalScrollBarPolicy(Qt::ScrollBarAlwaysOff);
    navScroll->setWidget(navigation);
    navScroll->setObjectName("navScroll");
    columns->addWidget(navScroll);
    auto center = new QWidget;
    auto centerLayout = new QVBoxLayout(center);
    centerLayout->setContentsMargins(0, 0, 0, 0);
    centerLayout->setSpacing(12);
    columns->addWidget(center, 1);
    composerPanel = card();
    auto compose = new QVBoxLayout(composerPanel);
    composer = new QTextEdit;
    composer->setAcceptRichText(false);
    composer->setPlaceholderText(tr("Có suy nghĩ gì…"));
    composer->setMaximumHeight(110);
    compose->addWidget(composer);
    stagedLabel = new QLabel;
    stagedLabel->setWordWrap(true);
    stagedLabel->hide();
    compose->addWidget(stagedLabel);
    auto actions = new QHBoxLayout;
    auto attach = button(tr("Attach files")), clearStage = button(tr("Clear files"));
    actions->addWidget(attach);
    actions->addWidget(clearStage);
    actions->addStretch();
    save = button(tr("Save"));
    save->setObjectName("primary");
    actions->addWidget(save);
    compose->addLayout(actions);
    centerLayout->addWidget(composerPanel);
    connect(attach, &QPushButton::clicked, this, [this] {
        const auto selected = QFileDialog::getOpenFileNames(this, tr("Attach files"));
        for (const auto &p : selected)
            if (!staged.contains(p))
                staged.append(p);
        QStringList names;
        for (const auto &p : staged)
            names.append(QFileInfo(p).fileName());
        stagedLabel->setText(names.join(" · "));
        stagedLabel->setVisible(!staged.isEmpty());
    });
    connect(clearStage, &QPushButton::clicked, this, [this] {
        staged.clear();
        stagedLabel->hide();
    });
    connect(save, &QPushButton::clicked, this, [this] {
        if (composer->toPlainText().trimmed().isEmpty() && staged.isEmpty())
            return;
        const auto text = composer->toPlainText();
        const auto files = staged;
        composerPanel->setEnabled(false);
        backend.call(this, "memos.create", {{"content", text}},
                     [this, files](const QJsonValue &value, const QString &code, const QString &message) {
                         composerPanel->setEnabled(true);
                         if (!code.isEmpty()) {
                             showError(message);
                             return;
                         }
                         composer->clear();
                         staged.clear();
                         stagedLabel->hide();
                         const auto uid = value.toObject()["uid"].toString();
                         if (files.isEmpty()) {
                             refresh();
                             metadata();
                         } else
                             addFiles(uid, files, [this] {
                                 refresh();
                                 metadata();
                             });
                     });
    });
    auto titleRow = new QHBoxLayout;
    heading = new QLabel(tr("All Notes"));
    heading->setObjectName("heading");
    titleRow->addWidget(heading);
    titleRow->addStretch();
    auto clear = button(tr("Clear filters"));
    titleRow->addWidget(clear);
    centerLayout->addLayout(titleRow);
    connect(clear, &QPushButton::clicked, this, [this] {
        tag.clear();
        date.clear();
        search->clear();
        navigate("home");
    });
    scroll = new QScrollArea;
    scroll->setWidgetResizable(true);
    scroll->setFrameShape(QFrame::NoFrame);
    rows = new QWidget;
    rowLayout = new QVBoxLayout(rows);
    rowLayout->setContentsMargins(0, 0, 4, 0);
    rowLayout->setSpacing(12);
    rowLayout->setAlignment(Qt::AlignTop);
    scroll->setWidget(rows);
    rows->setAutoFillBackground(false);
    scroll->viewport()->setAutoFillBackground(false);
    centerLayout->addWidget(scroll, 1);
    more = button(tr("Load more"));
    more->hide();
    centerLayout->addWidget(more, 0, Qt::AlignHCenter);
    connect(more, &QPushButton::clicked, this, [this] { refresh(true); });
    rightPanel = new QFrame;
    rightPanel->setObjectName("navigation");
    rightPanel->setFixedWidth(290);
    auto side = new QVBoxLayout(rightPanel);
    calendar = new QCalendarWidget;
    calendar->setVerticalHeaderFormat(QCalendarWidget::NoVerticalHeader);
    calendar->setGridVisible(false);
    calendar->setFirstDayOfWeek(Qt::Monday);
    side->addWidget(calendar);
    connect(calendar, &QCalendarWidget::clicked, this, [this](QDate selected) {
        date = selected.toString(Qt::ISODate);
        page = "home";
        refresh();
    });
    connect(calendar, &QCalendarWidget::currentPageChanged, this,
            [this] { refreshCalendar(calendar); });
    for (const auto &filter : QStringList{"All Notes", "Favorites", "Shared"}) {
        auto b = button(filter);
        side->addWidget(b);
        connect(b, &QPushButton::clicked, this, [this, filter] {
            tag.clear();
            date.clear();
            search->clear();
            navigate(filter == "All Notes" ? "home" : filter.toLower());
        });
    }
    side->addStretch();
    columns->addWidget(rightPanel);
    compact = settings.value("navigation/compact", false).toBool();
    connect(leftToggle, &QPushButton::clicked, this, [this] {
        compact = !compact;
        settings.setValue("navigation/compact", compact);
        applyNavigation();
    });
    debounce.setSingleShot(true);
    debounce.setInterval(300);
    connect(search, &QLineEdit::textChanged, this, [this] { debounce.start(); });
    connect(&debounce, &QTimer::timeout, this,
            [this] { navigate(search->text().isEmpty() ? "home" : "search"); });
    auto focus = new QShortcut(QKeySequence("Ctrl+K"), this);
    connect(focus, &QShortcut::activated, search, qOverload<>(&QWidget::setFocus));
    connect(&backend, &BackendClient::connected, this, [this](const QJsonObject &hello) {
        dataPath = hello["dataDir"].toString();
        save->setEnabled(true);
        statusBar()->showMessage(
            tr("Ready · Qt %1 · Go core %2").arg(QT_VERSION_STR, hello["version"].toString()));
        metadata();
        refresh();
    });
    connect(&backend, &BackendClient::failed, this, [this](const QString &message) {
        save->setEnabled(false);
        statusBar()->showMessage(tr("Backend disconnected — drafts kept"));
        showError(message);
    });
    save->setEnabled(false);
    applyAppearance();
    applyNavigation();
    QTimer::singleShot(0, this, [this, program, profile] { backend.start(program, profile); });
}
void MainWindow::showError(const QString &message) {
    if (activeError) {
        if (activeError->text() != message)
            activeError->setDetailedText(message);
        return;
    }
    activeError = new QMessageBox(QMessageBox::Warning, tr("NoteHub"), message, QMessageBox::Ok, this);
    activeError->setAttribute(Qt::WA_DeleteOnClose);
    activeError->open();
}
void MainWindow::rpc(const QString &method, const QJsonObject &params,
                     std::function<void(const QJsonValue &)> done) {
    backend.call(this, method, params,
                 [this, done](const QJsonValue &v, const QString &code, const QString &message) {
                     if (!code.isEmpty()) {
                         showError(message);
                         return;
                     }
                     if (done)
                         done(v);
                 });
}
void MainWindow::clearRows() {
    while (auto item = rowLayout->takeAt(0)) {
        if (auto w = item->widget())
            w->deleteLater();
        delete item;
    }
}
void MainWindow::navigate(const QString &destination) {
    debounce.stop();
    if (destination != "search") {
        QSignalBlocker blocker(search);
        search->clear();
    }
    page = destination;
    tag.clear();
    date.clear();
    cursor.clear();
    searchOffset = 0;
    attachmentOffset = 0;
    applyNavigation();
    refresh();
}
void MainWindow::applyNavigation() {
    const bool small = compact || width() < 900;
    navigationScroll->setFixedWidth(small ? 66 : 206);
    leftToggle->setText(small ? "◨" : "◧");
    leftToggle->setToolTip(small ? tr("Expand navigation") : tr("Collapse navigation"));
    for (auto b : navButtons) {
        b->setToolButtonStyle(small ? Qt::ToolButtonIconOnly : Qt::ToolButtonTextBesideIcon);
        b->setChecked(b->property("label").toString().toLower() == page);
    }
    rightPanel->setVisible(width() >= 1120);
}
void MainWindow::resizeEvent(QResizeEvent *e) {
    QMainWindow::resizeEvent(e);
    applyNavigation();
}
void MainWindow::metadata() {
    rpc("metadata", {}, [this](const QJsonValue &v) {
        auto m = v.toObject();
        tags = m["tags"].toArray();
        statusBar()->showMessage(tr("%1 notes · %2 favorites · %3 shared")
                                     .arg(m["all"].toInt())
                                     .arg(m["favorites"].toInt())
                                     .arg(m["shared"].toInt()));
    });
    refreshCalendar(calendar);
}
void MainWindow::refreshCalendar(QCalendarWidget *widget) {
    if (!backend.isReady())
        return;
    const int year = widget->yearShown(), month = widget->monthShown();
    backend.call(widget, "calendar.month", {{"year", year}, {"month", month}},
        [widget, year, month](const QJsonValue &v, const QString &code, const QString &) {
            if (!code.isEmpty() || widget->yearShown() != year || widget->monthShown() != month)
                return;
            widget->setDateTextFormat(QDate(), QTextCharFormat());
            for (const auto &d : v.toArray()) {
                auto day = d.toObject();
                QTextCharFormat f;
                f.setFontWeight(QFont::Bold);
                f.setForeground(QColor("#087bff"));
                f.setToolTip(QObject::tr("%1 notes").arg(day["count"].toInt()));
                widget->setDateTextFormat(QDate::fromString(day["date"].toString(), Qt::ISODate), f);
            }
        });
}
void MainWindow::refresh(bool append) {
    if (!backend.isReady())
        return;
    if (append && loading)
        return;
    applyNavigation();
    const int current = ++generation;
    loading = true;
    more->hide();
    composerPanel->setVisible(page == "home");
    heading->setText(!tag.isEmpty()    ? "#" + tag
                     : !date.isEmpty() ? date
                                       : page.left(1).toUpper() + page.mid(1));
    if (!append) {
        cursor.clear();
        searchOffset = 0;
        attachmentOffset = 0;
        clearRows();
    }
    if (page == "settings") {
        loading = false;
        settingsPage();
        return;
    }
    if (page == "calendar") {
        loading = false;
        auto c = new QCalendarWidget;
        c->setMinimumHeight(360);
        c->setFirstDayOfWeek(Qt::Monday);
        c->setVerticalHeaderFormat(QCalendarWidget::NoVerticalHeader);
        rowLayout->addWidget(c);
        connect(c, &QCalendarWidget::currentPageChanged, this, [this, c] { refreshCalendar(c); });
        refreshCalendar(c);
        connect(c, &QCalendarWidget::clicked, this, [this](QDate d) {
            date = d.toString(Qt::ISODate);
            page = "home";
            refresh();
        });
        return;
    }
    if (page == "tags") {
        loading = false;
        rpc("metadata", {}, [this, current](const QJsonValue &v) {
            if (current != generation)
                return;
            tags = v.toObject()["tags"].toArray();
            for (const auto &item : tags) {
                auto t = item.toObject();
                auto b = button("#" + t["tag"].toString() + "  ·  " + QString::number(t["count"].toInt()));
                rowLayout->addWidget(b);
                connect(b, &QPushButton::clicked, this, [this, t] {
                    tag = t["tag"].toString();
                    page = "home";
                    refresh();
                });
            }
        });
        return;
    }
    if (page == "attachments") {
        backend.call(this, "attachments.list", {{"limit", 40}, {"offset", attachmentOffset}},
                     [this, current](const QJsonValue &v, const QString &code, const QString &message) {
                         if (current != generation)
                             return;
                         loading = false;
                         if (!code.isEmpty()) {
                             showError(message);
                             return;
                         }
                         auto values = v.toArray();
                         attachmentOffset += values.size();
                         more->setVisible(values.size() == 40);
                         for (const auto &item : values) {
                             auto a = item.toObject();
                             auto f = card();
                             auto l = new QHBoxLayout(f);
                             auto name = button(a["filename"].toString());
                             name->setToolTip(a["path"].toString());
                             l->addWidget(name, 1);
                             auto note = button(tr("Note")), remove = button(tr("Delete"));
                             l->addWidget(note);
                             l->addWidget(remove);
                             rowLayout->addWidget(f);
                             connect(name, &QPushButton::clicked, this, [this, a] {
                                 if (!QDesktopServices::openUrl(QUrl::fromLocalFile(a["path"].toString())))
                                     showError(tr("Could not open attachment."));
                             });
                             connect(note, &QPushButton::clicked, this,
                                     [this, a] { editNote(a["memoUid"].toString()); });
                             connect(remove, &QPushButton::clicked, this, [this, a] {
                                 if (QMessageBox::question(this, tr("Delete attachment?"),
                                                           a["filename"].toString()) == QMessageBox::Yes)
                                     rpc("attachments.delete", {{"uid", a["uid"]}},
                                         [this](const QJsonValue &) { refresh(); });
                             });
                         }
                     });
        return;
    }
    QJsonObject p{{"limit", 20},
                  {"cursor", cursor},
                  {"offset", searchOffset},
                  {"favorite", page == "favorites"},
                  {"shared", page == "shared"}};
    if (!tag.isEmpty())
        p["tags"] = QJsonArray{tag};
    if (!date.isEmpty())
        p["date"] = date;
    if (page == "search")
        p["text"] = search->text();
    backend.call(
        this, "memos.list", p,
        [this, current, append](const QJsonValue &v, const QString &code, const QString &message) {
            if (current != generation)
                return;
            loading = false;
            if (!code.isEmpty()) {
                showError(message);
                return;
            }
            auto result = v.toObject();
            auto items = result["items"].toArray();
            cursor = result["cursor"].toString();
            searchOffset = result["nextOffset"].toInt();
            more->setVisible(!cursor.isEmpty() || searchOffset > 0);
            for (const auto &item : items)
                addNote(item.toObject());
            if (items.isEmpty() && !append) {
                auto empty = new QLabel(
                    tr("A little space for your thoughts\nWrite a note above or choose another filter."));
                empty->setAlignment(Qt::AlignCenter);
                empty->setMinimumHeight(160);
                rowLayout->addWidget(empty);
            }
        });
}
void MainWindow::addNote(const QJsonObject &memo) {
    auto f = card();
    auto l = new QVBoxLayout(f);
    l->setSpacing(10);
    auto top = new QHBoxLayout;
    auto stamp = new QLabel(QDateTime::fromString(memo["createdAt"].toString(), Qt::ISODate)
                                .toLocalTime()
                                .toString("dd MMM yyyy · HH:mm"));
    top->addWidget(stamp);
    top->addStretch();
    auto favorite = button(memo["favorite"].toBool() ? "★" : "☆"), edit = button(tr("Edit")),
         menu = button("···");
    top->addWidget(favorite);
    top->addWidget(edit);
    top->addWidget(menu);
    l->addLayout(top);
    const auto uid = memo["uid"].toString();
    connect(favorite, &QPushButton::clicked, this, [this, uid, memo] {
        rpc("memos.favorite", {{"uid", uid}, {"favorite", !memo["favorite"].toBool()}},
            [this](const QJsonValue &) {
                refresh();
                metadata();
            });
    });
    connect(edit, &QPushButton::clicked, this, [this, uid] { editNote(uid); });
    connect(menu, &QPushButton::clicked, this, [this, uid, menu] {
        QMenu m;
        auto share = m.addAction(tr("Share"));
        auto remove = m.addAction(tr("Delete"));
        auto chosen = m.exec(menu->mapToGlobal(QPoint(0, menu->height())));
        if (chosen == share)
            shareNote(uid);
        if (chosen == remove &&
            QMessageBox::question(this, tr("Delete note?"), tr("Delete this note and its attachments?")) ==
                QMessageBox::Yes)
            rpc("memos.delete", {{"uid", uid}}, [this](const QJsonValue &) {
                refresh();
                metadata();
            });
    });
    auto text = new QTextBrowser;
    text->setDocument(new LocalTextDocument(text));
    text->setOpenLinks(false);
    text->setFrameShape(QFrame::NoFrame);
    loadNoteDocument(text->document(), memo["content"].toString());
    text->setMinimumHeight(64);
    text->setMaximumHeight(150);
    text->setSizePolicy(QSizePolicy::Expanding, QSizePolicy::Preferred);
    l->addWidget(text);
    const auto files = memo["attachments"].toArray();
    bool images = false;
    for (const auto &a : files)
        images |= a.toObject()["mimeType"].toString().startsWith("image/");
    if (images) {
        auto gallery = new ImageGallery(files);
        l->addWidget(gallery);
        connect(gallery, &ImageGallery::opened, this,
                [](const QString &p) { QDesktopServices::openUrl(QUrl::fromLocalFile(p)); });
        connect(gallery, &ImageGallery::reordered, this, [this, uid, gallery](const QJsonArray &order) {
            gallery->setEnabled(false);
            QPointer<ImageGallery> guard(gallery);
            backend.call(this, "attachments.reorder", {{"uid", uid}, {"order", order}},
                         [this, guard](const QJsonValue &, const QString &code, const QString &message) {
                             if (guard)
                                 guard->setEnabled(true);
                             if (!code.isEmpty())
                                 showError(message);
                             else
                                 refresh();
                         });
        });
    }
    for (const auto &value : files) {
        auto a = value.toObject();
        if (a["mimeType"].toString().startsWith("image/"))
            continue;
        auto b = button(a["filename"].toString());
        l->addWidget(b);
        connect(b, &QPushButton::clicked, this,
                [a] { QDesktopServices::openUrl(QUrl::fromLocalFile(a["path"].toString())); });
    }
    if (!memo["tags"].toArray().isEmpty()) {
        auto labels = new QLabel;
        QStringList names;
        for (const auto &t : memo["tags"].toArray())
            names << "#" + t.toString();
        labels->setText(names.join("  "));
        labels->setWordWrap(true);
        l->addWidget(labels);
    }
    rowLayout->addWidget(f);
}
void MainWindow::addFiles(const QString &uid, const QStringList &paths, std::function<void()> done) {
    rpc("attachments.add", {{"uid", uid}, {"paths", jsonPaths(paths)}}, [this, done](const QJsonValue &v) {
        auto failures = fileFailures(v);
        if (!failures.isEmpty())
            showError(tr("The note was saved, but some attachments failed:\n") + failures.join('\n'));
        if (done)
            done();
    });
}

void MainWindow::editNote(const QString &uid) {
    rpc("memos.get", {{"uid", uid}}, [this, uid](const QJsonValue &v) {
        auto memo = v.toObject();
        auto dialog = new MemoDialog(this);
        dialog->setAttribute(Qt::WA_DeleteOnClose);
        dialog->setWindowTitle(tr("Edit Note"));
        dialog->resize(qMin(940, width() - 60), qMin(780, height() - 60));
        dialog->setMinimumSize(500, 360);
        auto l = new QVBoxLayout(dialog);
        auto editor = new NoteEditor;
        dialog->editor = editor;
        editor->setContent(memo["content"].toString());
        l->addWidget(editor, 1);
        auto attachments = new QListWidget;
        attachments->setMaximumHeight(100);
        l->addWidget(attachments);
        auto fill = [attachments](const QJsonObject &m) {
            attachments->clear();
            for (const auto &v : m["attachments"].toArray()) {
                auto a = v.toObject();
                auto item = new QListWidgetItem(a["filename"].toString(), attachments);
                item->setData(Qt::UserRole, a);
            }
        };
        fill(memo);
        auto files = new QHBoxLayout;
        auto attach = button(tr("Attach files")), remove = button(tr("Remove selected"));
        files->addWidget(attach);
        files->addWidget(remove);
        files->addStretch();
        l->addLayout(files);
        auto status = new QLabel;
        status->setWordWrap(true);
        l->addWidget(status);
        auto buttons = new QHBoxLayout;
        auto reload = button(tr("Reload latest")), cancel = button(tr("Cancel")),
             saveNote = button(tr("Save"));
        saveNote->setObjectName("primary");
        buttons->addWidget(reload);
        buttons->addStretch();
        buttons->addWidget(cancel);
        buttons->addWidget(saveNote);
        l->addLayout(buttons);
        auto current = std::make_shared<QJsonObject>(memo);
        auto setBusy = [dialog, editor, attach, remove, reload, cancel, saveNote](bool busy) {
            dialog->busy = busy;
            for (auto w : QList<QWidget *>{editor, attach, remove, reload, cancel, saveNote})
                w->setEnabled(!busy);
        };
        connect(cancel, &QPushButton::clicked, dialog, &QDialog::reject);
        connect(
            saveNote, &QPushButton::clicked, dialog, [this, dialog, editor, status, current, uid, setBusy] {
                setBusy(true);
                status->setText(tr("Saving…"));
                backend.call(
                    dialog, "memos.update",
                    {{"uid", uid}, {"revision", (*current)["revision"]}, {"content", editor->content()}},
                    [this, dialog, status, setBusy](const QJsonValue &, const QString &code,
                                                    const QString &message) {
                        setBusy(false);
                        if (!code.isEmpty()) {
                            status->setText(code == "conflict" ? tr("This note changed elsewhere. Your draft "
                                                                    "is kept. Copy it before reloading.")
                                                               : message);
                            return;
                        }
                        dialog->accept();
                        refresh();
                        metadata();
                    });
            });
        connect(reload, &QPushButton::clicked, dialog,
                [this, dialog, editor, status, current, uid, setBusy, fill] {
                    if (editor->modified() &&
                        QMessageBox::question(dialog, tr("Reload note?"),
                                              tr("Replace your unsaved draft with the latest saved note?")) !=
                            QMessageBox::Yes)
                        return;
                    setBusy(true);
                    backend.call(dialog, "memos.get", {{"uid", uid}},
                                 [editor, status, current, setBusy,
                                  fill](const QJsonValue &v, const QString &code, const QString &message) {
                                     setBusy(false);
                                     if (!code.isEmpty()) {
                                         status->setText(message);
                                         return;
                                     }
                                     *current = v.toObject();
                                     editor->setContent((*current)["content"].toString());
                                     fill(*current);
                                     status->clear();
                                 });
                });
        connect(attach, &QPushButton::clicked, dialog, [this, dialog, status, current, uid, setBusy, fill] {
            auto paths = QFileDialog::getOpenFileNames(dialog, tr("Attach files"));
            if (paths.isEmpty())
                return;
            setBusy(true);
            backend.call(dialog, "attachments.add", {{"uid", uid}, {"paths", jsonPaths(paths)}},
                         [this, status, current, setBusy, fill](const QJsonValue &v, const QString &code,
                                                                const QString &message) {
                             setBusy(false);
                             if (!code.isEmpty()) {
                                 status->setText(message);
                                 return;
                             }
                             (*current)["attachments"] = v.toObject()["memo"].toObject()["attachments"];
                             fill(*current);
                             status->setText(fileFailures(v).join('\n'));
                             refresh();
                         });
        });
        connect(remove, &QPushButton::clicked, dialog, [this, dialog, status, attachments, current, setBusy] {
            auto item = attachments->currentItem();
            if (!item)
                return;
            auto a = item->data(Qt::UserRole).toJsonObject();
            if (QMessageBox::question(dialog, tr("Remove attachment?"), a["filename"].toString()) !=
                QMessageBox::Yes)
                return;
            setBusy(true);
            backend.call(dialog, "attachments.delete", {{"uid", a["uid"]}},
                         [this, status, attachments, current, a,
                          setBusy](const QJsonValue &, const QString &code, const QString &message) {
                             setBusy(false);
                             if (!code.isEmpty()) {
                                 status->setText(message);
                                 return;
                             }
                             auto values = (*current)["attachments"].toArray();
                             for (int i = 0; i < values.size(); ++i)
                                 if (values[i].toObject()["uid"] == a["uid"]) {
                                     values.removeAt(i);
                                     delete attachments->takeItem(i);
                                     break;
                                 }
                             (*current)["attachments"] = values;
                             refresh();
                         });
        });
        dialog->open();
    });
}
void MainWindow::shareNote(const QString &uid) {
    auto dialog = new QDialog(this);
    dialog->setAttribute(Qt::WA_DeleteOnClose);
    dialog->setWindowTitle(tr("Share note"));
    dialog->resize(560, 360);
    auto l = new QVBoxLayout(dialog);
    auto hint = new QLabel(tr("Links work on this computer while local sharing is enabled in Settings."));
    hint->setWordWrap(true);
    l->addWidget(hint);
    auto list = new QListWidget;
    l->addWidget(list);
    auto expiry = new QComboBox;
    expiry->addItems({tr("No expiry"), tr("1 day"), tr("7 days"), tr("30 days")});
    l->addWidget(expiry);
    auto url = new QLineEdit;
    url->setReadOnly(true);
    url->setPlaceholderText(tr("New link appears here once — copy it now"));
    l->addWidget(url);
    auto create = button(tr("Create link")), revoke = button(tr("Revoke selected"));
    l->addWidget(create);
    l->addWidget(revoke);
    auto load = [this, dialog, list, uid] {
        backend.call(dialog, "shares.list", {{"uid", uid}},
                     [this, list](const QJsonValue &v, const QString &code, const QString &message) {
                         if (!code.isEmpty()) {
                             showError(message);
                             return;
                         }
                         list->clear();
                         for (const auto &item : v.toArray()) {
                             auto s = item.toObject();
                             auto row = new QListWidgetItem(
                                 s["uid"].toString() + " · " +
                                     (s["expiresAt"].isNull() ? tr("No expiry") : s["expiresAt"].toString()),
                                 list);
                             row->setData(Qt::UserRole, s["uid"].toString());
                         }
                     });
    };
    connect(create, &QPushButton::clicked, dialog, [this, dialog, uid, expiry, url, load] {
        const int days[] = {0, 1, 7, 30};
        backend.call(dialog, "shares.create", {{"uid", uid}, {"days", days[expiry->currentIndex()]}},
                     [this, url, load](const QJsonValue &v, const QString &code, const QString &message) {
                         if (!code.isEmpty()) {
                             showError(message);
                             return;
                         }
                         url->setText(v.toObject()["url"].toString());
                         url->selectAll();
                         load();
                         metadata();
                     });
    });
    connect(revoke, &QPushButton::clicked, dialog, [this, dialog, list, load] {
        if (!list->currentItem())
            return;
        backend.call(dialog, "shares.revoke", {{"uid", list->currentItem()->data(Qt::UserRole).toString()}},
                     [this, load](const QJsonValue &, const QString &code, const QString &message) {
                         if (!code.isEmpty())
                             showError(message);
                         else {
                             load();
                             metadata();
                         }
                     });
    });
    load();
    dialog->open();
}
void MainWindow::settingsPage() {
    auto f = card();
    auto form = new QFormLayout(f);
    auto appearance = new QComboBox;
    appearance->addItems({"System", "Light", "Dark"});
    appearance->setCurrentText(settings.value("appearance", "Light").toString());
    form->addRow(tr("Appearance"), appearance);
    connect(appearance, &QComboBox::currentTextChanged, this, [this](const QString &v) {
        settings.setValue("appearance", v);
        applyAppearance();
    });
    auto font = new QFontComboBox;
    font->setCurrentFont(QFont(settings.value("fontFamily", qApp->font().family()).toString()));
    form->addRow(tr("Font"), font);
    connect(font, &QFontComboBox::currentFontChanged, this, [this](const QFont &f) {
        settings.setValue("fontFamily", f.family());
        applyAppearance();
    });
    auto size = new QSpinBox;
    size->setRange(9, 24);
    size->setValue(settings.value("fontSize", 11).toInt());
    form->addRow(tr("Text size"), size);
    connect(size, &QSpinBox::valueChanged, this, [this](int n) {
        settings.setValue("fontSize", n);
        applyAppearance();
    });
    auto folder = button(tr("Open data folder"));
    form->addRow(folder);
    connect(folder, &QPushButton::clicked, this,
            [this] { QDesktopServices::openUrl(QUrl::fromLocalFile(dataPath)); });
    auto exportFile = button(tr("Export ZIP backup")), importFile = button(tr("Import ZIP backup"));
    form->addRow(exportFile);
    form->addRow(importFile);
    connect(exportFile, &QPushButton::clicked, this, [this] {
        auto path = QFileDialog::getSaveFileName(
            this, tr("Export to a new file"),
            "NoteHub-" + QDate::currentDate().toString(Qt::ISODate) + ".zip", tr("Backup (*.zip)"));
        if (path.isEmpty())
            return;
        rpc("backup.export", {{"path", path}}, [this](const QJsonValue &) {
            QMessageBox::information(this, tr("Backup"), tr("Backup exported."));
        });
    });
    connect(importFile, &QPushButton::clicked, this, [this] {
        auto path = QFileDialog::getOpenFileName(this, tr("Import backup"), {}, tr("Backup (*.zip)"));
        if (path.isEmpty())
            return;
        bool ok;
        auto policy = QInputDialog::getItem(this, tr("Conflicting notes"), tr("Import policy"),
                                            {"skip", "replace", "duplicate"}, 0, false, &ok);
        if (!ok)
            return;
        rpc("backup.import", {{"path", path}, {"policy", policy}}, [this](const QJsonValue &v) {
            QMessageBox::information(
                this, tr("Import result"),
                QString::fromUtf8(QJsonDocument(v.toObject()).toJson(QJsonDocument::Indented)));
            metadata();
            refresh();
        });
    });
    auto sharing = new QCheckBox(tr("Enable local sharing"));
    auto port = new QSpinBox;
    port->setRange(0, 65535);
    port->setValue(settings.value("sharingPort", 8787).toInt());
    port->setSpecialValueText(tr("Automatic port"));
    form->addRow(sharing);
    form->addRow(tr("Sharing port"), port);
    QPointer<QCheckBox> guard(sharing);
    backend.call(sharing, "metadata", {}, [guard](const QJsonValue &v, const QString &code, const QString &) {
        if (guard && code.isEmpty()) {
            QSignalBlocker b(guard);
            guard->setChecked(!v.toObject()["sharingUrl"].toString().isEmpty());
        }
    });
    connect(sharing, &QCheckBox::toggled, this, [this, sharing, port](bool enabled) {
        sharing->setEnabled(false);
        settings.setValue("sharingPort", port->value());
        QPointer<QCheckBox> guard(sharing);
        backend.call(sharing, "sharing.set", {{"enabled", enabled}, {"port", port->value()}},
                     [this, guard, enabled](const QJsonValue &, const QString &code, const QString &message) {
                         if (!guard)
                             return;
                         guard->setEnabled(true);
                         if (!code.isEmpty()) {
                             QSignalBlocker b(guard);
                             guard->setChecked(!enabled);
                             showError(message);
                         }
                         metadata();
                     });
    });
    form->addRow(new QLabel(
        tr("NoteHub %1 · Qt GUI + Go backend\nSharing is off on every launch.").arg(NOTEHUB_VERSION)));
    rowLayout->addWidget(f);
}
void MainWindow::applyAppearance() {
    const auto mode = settings.value("appearance", "Light").toString();
    bool dark = mode == "Dark" ||
                (mode == "System" && QGuiApplication::styleHints()->colorScheme() == Qt::ColorScheme::Dark);
    QFont font(settings.value("fontFamily", qApp->font().family()).toString(),
               settings.value("fontSize", 11).toInt());
    qApp->setFont(font);
    QPalette p = qApp->style()->standardPalette();
    if (dark) {
        p.setColor(QPalette::Window, QColor("#101823"));
        p.setColor(QPalette::WindowText, QColor("#e4ebf5"));
        p.setColor(QPalette::Base, QColor("#172333"));
        p.setColor(QPalette::AlternateBase, QColor("#203147"));
        p.setColor(QPalette::Text, QColor("#e4ebf5"));
        p.setColor(QPalette::Button, QColor("#233348"));
        p.setColor(QPalette::ButtonText, QColor("#e4ebf5"));
        p.setColor(QPalette::Mid, QColor("#35475f"));
        p.setColor(QPalette::PlaceholderText, QColor("#96a6bd"));
    }
    p.setColor(QPalette::Highlight, QColor("#087bff"));
    qApp->setPalette(p);
    qApp->setStyleSheet(
        QString("QMainWindow{background:%1} QFrame#card{background:%2;border:1px solid "
                "%3;border-radius:12px} QFrame#navigation{background:%4;border-radius:12px} "
                "QLabel#brand{font-size:25px;font-weight:700} QLabel#heading{font-size:18px;font-weight:600} "
                "QPushButton,QToolButton{padding:8px;border:0;border-radius:7px} "
                "QPushButton:hover,QToolButton:hover{background:%4} "
                "QToolButton:checked{background:%5;color:#087bff} "
                "QPushButton#primary{background:#087bff;color:white;font-weight:600} "
                "QLineEdit,QTextEdit,QPlainTextEdit{border:1px solid %3;border-radius:8px;padding:8px} "
                "QTextBrowser{border:0;padding:0;background:transparent} "
                "QScrollArea{border:0;background:transparent} QToolBar{border:0;spacing:2px} QToolBar "
                "QToolButton{padding:5px}")
            .arg(dark ? "#101823" : "#f7f9fc", dark ? "#172333" : "#ffffff", dark ? "#35475f" : "#e0e7f0",
                 dark ? "#203147" : "#edf3fa", dark ? "#253d5c" : "#dfeeff"));
}
void MainWindow::closeEvent(QCloseEvent *e) {
    if (backend.busy()) {
        QMessageBox::information(
            this, tr("Operation in progress"),
            tr("Please wait for the current save, import or export to finish before closing."));
        e->ignore();
        return;
    }
    if ((!composer->toPlainText().trimmed().isEmpty() || !staged.isEmpty()) &&
        QMessageBox::question(this, tr("Discard draft?"),
                              tr("The Home draft has not been saved. Close anyway?")) != QMessageBox::Yes) {
        e->ignore();
        return;
    }
    for (auto child : findChildren<QDialog *>())
        if (auto dialog = dynamic_cast<MemoDialog *>(child))
            if (dialog->isVisible() && dialog->editor && dialog->editor->modified()) {
                dialog->raise();
                e->ignore();
                return;
            }
    backend.stop();
    e->accept();
}
